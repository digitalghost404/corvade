package rules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/corvade/corvade/internal/policy"
)

// PIIRedactionRule scans request bodies for PII patterns and either blocks or redacts matches.
type PIIRedactionRule struct {
	name         string
	mode         string
	patterns     []*regexp.Regexp
	patternNames []string
	action       string // block | redact
}

// EvalResultWithModified holds both a violation and optional modified body for redaction.
type EvalResultWithModified struct {
	Violation *policy.Violation
	Modified  []byte
}

var defaultPIIPatterns = []struct {
	name    string
	pattern string
}{
	{"email", `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`},
	{"phone", `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`},
	{"ssn", `\b\d{3}-\d{2}-\d{4}\b`},
	{"credit_card", `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`},
}

// NewPIIRedactionRule constructs a PIIRedactionRule from a raw config map.
// Uses built-in PII patterns. Config may specify "action" as "block" (default) or "redact".
func NewPIIRedactionRule(name, mode string, config map[string]interface{}) (*PIIRedactionRule, error) {
	action := "block"
	if raw, ok := config["action"]; ok {
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("pii_redaction rule %q: 'action' must be a string, got %T", name, raw)
		}
		switch s {
		case "block", "redact":
			action = s
		default:
			return nil, fmt.Errorf("pii_redaction rule %q: 'action' must be 'block' or 'redact', got %q", name, s)
		}
	}

	patterns := make([]*regexp.Regexp, 0, len(defaultPIIPatterns))
	patternNames := make([]string, 0, len(defaultPIIPatterns))
	for _, p := range defaultPIIPatterns {
		re, err := regexp.Compile(p.pattern)
		if err != nil {
			return nil, fmt.Errorf("pii_redaction rule %q: failed to compile pattern %q: %w", name, p.name, err)
		}
		patterns = append(patterns, re)
		patternNames = append(patternNames, p.name)
	}

	return &PIIRedactionRule{
		name:         name,
		mode:         mode,
		patterns:     patterns,
		patternNames: patternNames,
		action:       action,
	}, nil
}

func (r *PIIRedactionRule) Name() string { return r.name }
func (r *PIIRedactionRule) Type() string { return "pii_redaction" }
func (r *PIIRedactionRule) Mode() string { return r.mode }

// Evaluate checks the request body for PII and returns a Violation if found.
func (r *PIIRedactionRule) Evaluate(ctx policy.EvalContext) *policy.Violation {
	result := r.EvaluateWithResult(ctx)
	if result == nil {
		return nil
	}
	return result.Violation
}

// EvaluateWithPIIResult implements policy.PIIRedactor, returning a PIIResult
// that the engine can use to access the modified body without importing this package.
func (r *PIIRedactionRule) EvaluateWithPIIResult(ctx policy.EvalContext) *policy.PIIResult {
	ewm := r.evaluateInternal(ctx)
	if ewm == nil {
		return nil
	}
	return &policy.PIIResult{
		Violation: ewm.Violation,
		Modified:  ewm.Modified,
	}
}

// EvaluateWithResult returns both a violation and optional modified body (for redaction).
func (r *PIIRedactionRule) EvaluateWithResult(ctx policy.EvalContext) *EvalResultWithModified {
	return r.evaluateInternal(ctx)
}

// evaluateInternal is the shared implementation for both result methods.
func (r *PIIRedactionRule) evaluateInternal(ctx policy.EvalContext) *EvalResultWithModified {
	if len(ctx.RequestBody) == 0 {
		return nil
	}

	text := extractTextContent(ctx.RequestBody)
	if text == "" {
		return nil
	}

	// Count matches per pattern
	type matchInfo struct {
		name  string
		count int
	}
	var matches []matchInfo

	for i, re := range r.patterns {
		found := re.FindAllString(text, -1)
		if len(found) > 0 {
			matches = append(matches, matchInfo{name: r.patternNames[i], count: len(found)})
		}
	}

	if len(matches) == 0 {
		return nil
	}

	// Build message
	var parts []string
	for _, m := range matches {
		noun := m.name
		if m.count > 1 {
			noun += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", m.count, noun))
	}
	msg := fmt.Sprintf("PII detected: %s found in request", strings.Join(parts, ", "))

	violation := &policy.Violation{
		Rule:    r.name,
		Type:    r.Type(),
		Mode:    r.mode,
		Message: msg,
	}

	result := &EvalResultWithModified{Violation: violation}

	// Redaction: replace matches in the full body string
	if r.action == "redact" && r.mode == "enforce" {
		modified := string(ctx.RequestBody)
		for _, re := range r.patterns {
			modified = re.ReplaceAllString(modified, "[REDACTED]")
		}
		result.Modified = []byte(modified)
	}

	return result
}

// extractTextContent pulls text from a JSON request body's messages array.
func extractTextContent(body []byte) string {
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}

	var sb strings.Builder
	for _, m := range req.Messages {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(m.Content)
	}
	return sb.String()
}
