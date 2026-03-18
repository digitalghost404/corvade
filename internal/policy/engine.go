package policy

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"gopkg.in/yaml.v3"
)

// RuleBuilderFunc constructs a Rule from a RuleConfig.
// The store parameter is passed for rules that need database access (e.g. cost_ceiling).
type RuleBuilderFunc func(rc RuleConfig, store *capture.Store) (Rule, error)

// PIIRedactor is implemented by rules that can return a modified (redacted) body
// alongside a violation. This allows the engine to handle PII redaction without
// importing the rules package directly.
type PIIRedactor interface {
	Rule
	EvaluateWithPIIResult(ctx EvalContext) *PIIResult
}

// PIIResult holds both a violation and optional modified body for redaction rules.
type PIIResult struct {
	Violation *Violation
	Modified  []byte
}

// Engine loads YAML policies, watches for file changes, and evaluates rules.
type Engine struct {
	policyPath string
	store      *capture.Store
	calc       *cost.Calculator
	buildRule  RuleBuilderFunc
	rules      []Rule
	webhook    *WebhookConfig
	mu         sync.RWMutex
	modTime    time.Time
	stopCh     chan struct{}
}

// NewEngine creates an engine and loads policies from the given path.
// If the file does not exist, the engine starts with zero rules (no error).
// If the file exists but is invalid, an error is returned.
// The builder parameter constructs Rule implementations from config; use
// rules.DefaultBuilder() from the policy/rules package.
func NewEngine(policyPath string, store *capture.Store, calc *cost.Calculator, opts ...EngineOption) (*Engine, error) {
	e := &Engine{
		policyPath: policyPath,
		store:      store,
		calc:       calc,
		stopCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.buildRule == nil {
		return nil, fmt.Errorf("policy engine: no rule builder configured (use WithRuleBuilder option)")
	}

	if err := e.loadPolicies(); err != nil {
		return nil, err
	}
	return e, nil
}

// EngineOption configures the Engine.
type EngineOption func(*Engine)

// WithRuleBuilder sets the function used to construct Rule implementations from config.
func WithRuleBuilder(b RuleBuilderFunc) EngineOption {
	return func(e *Engine) {
		e.buildRule = b
	}
}

// loadPolicies reads and parses the YAML policy file, constructing Rule instances.
func (e *Engine) loadPolicies() error {
	data, err := os.ReadFile(e.policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			e.mu.Lock()
			e.rules = nil
			e.webhook = nil
			e.mu.Unlock()
			return nil
		}
		return fmt.Errorf("reading policy file: %w", err)
	}

	var pf PolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return fmt.Errorf("parsing policy file: %w", err)
	}

	constructed := make([]Rule, 0, len(pf.Rules))
	for _, rc := range pf.Rules {
		if rc.Mode != "observe" && rc.Mode != "enforce" {
			return fmt.Errorf("rule %q: invalid mode %q (must be 'observe' or 'enforce')", rc.Name, rc.Mode)
		}

		r, err := e.buildRule(rc, e.store)
		if err != nil {
			return err
		}
		constructed = append(constructed, r)
	}

	// Record mtime for watch detection
	info, _ := os.Stat(e.policyPath)
	var mt time.Time
	if info != nil {
		mt = info.ModTime()
	}

	e.mu.Lock()
	e.rules = constructed
	e.webhook = pf.Webhook
	e.modTime = mt
	e.mu.Unlock()

	return nil
}

// Evaluate runs all loaded rules against the given context and returns an EvalResult.
func (e *Engine) Evaluate(ctx EvalContext) EvalResult {
	e.mu.RLock()
	currentRules := e.rules
	e.mu.RUnlock()

	var result EvalResult

	for _, r := range currentRules {
		var v *Violation
		var modified []byte

		// Special handling for PII redaction which can return a modified body.
		if pii, ok := r.(PIIRedactor); ok {
			pr := pii.EvaluateWithPIIResult(ctx)
			if pr != nil {
				v = pr.Violation
				modified = pr.Modified
			}
		} else {
			v = r.Evaluate(ctx)
		}

		if v == nil {
			continue
		}

		result.Violations = append(result.Violations, *v)

		if modified != nil {
			result.Modified = modified
		}

		if v.Mode == "enforce" {
			result.Blocked = true
			break
		}
	}

	if len(result.Violations) > 0 && e.webhook != nil {
		go e.sendWebhook(result.Violations, ctx)
	}

	return result
}

// Start begins the file watcher goroutine that polls for policy changes.
func (e *Engine) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-e.stopCh:
				return
			case <-ticker.C:
				info, err := os.Stat(e.policyPath)
				if err != nil {
					continue
				}

				e.mu.RLock()
				prevMod := e.modTime
				e.mu.RUnlock()

				if info.ModTime().After(prevMod) {
					if err := e.loadPolicies(); err != nil {
						log.Printf("policy: hot reload failed: %v (keeping previous rules)", err)
					}
				}
			}
		}
	}()
}

// Stop shuts down the file watcher goroutine.
func (e *Engine) Stop() {
	close(e.stopCh)
}

// Rules returns a summary of all loaded rules.
func (e *Engine) Rules() []RuleSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	summaries := make([]RuleSummary, 0, len(e.rules))
	for _, r := range e.rules {
		summaries = append(summaries, RuleSummary{
			Name:  r.Name(),
			Type:  r.Type(),
			Mode:  r.Mode(),
			Valid: true,
		})
	}
	return summaries
}

// Validate loads and validates a policy file without starting an engine.
// Returns rule summaries on success or an error if the file is missing or invalid.
func Validate(policyPath string, builder RuleBuilderFunc) ([]RuleSummary, error) {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	var pf PolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing policy file: %w", err)
	}

	summaries := make([]RuleSummary, 0, len(pf.Rules))
	for _, rc := range pf.Rules {
		s := RuleSummary{
			Name: rc.Name,
			Type: rc.Type,
			Mode: rc.Mode,
		}

		if rc.Mode != "observe" && rc.Mode != "enforce" {
			s.Error = fmt.Sprintf("invalid mode %q", rc.Mode)
			summaries = append(summaries, s)
			return nil, fmt.Errorf("rule %q: invalid mode %q", rc.Name, rc.Mode)
		}

		if _, err := builder(rc, nil); err != nil {
			s.Error = err.Error()
			summaries = append(summaries, s)
			return nil, err
		}

		s.Valid = true
		summaries = append(summaries, s)
	}

	return summaries, nil
}
