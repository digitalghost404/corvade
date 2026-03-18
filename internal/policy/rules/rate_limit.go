package rules

import (
	"fmt"
	"sync"
	"time"

	"github.com/corvade/corvade/internal/policy"
)

// RateLimitRule enforces a sliding window rate limit on requests.
type RateLimitRule struct {
	name        string
	mode        string
	maxRequests int
	windowDur   time.Duration
	per         string // global | agent | model
	mu          sync.Mutex
	counters    map[string][]time.Time
}

// NewRateLimitRule constructs a RateLimitRule from a raw config map.
// Required config keys: "max_requests" (number), "window" (duration string), "per" (global|agent|model).
func NewRateLimitRule(name, mode string, config map[string]interface{}) (*RateLimitRule, error) {
	raw, ok := config["max_requests"]
	if !ok {
		return nil, fmt.Errorf("rate_limit rule %q: missing required config key 'max_requests'", name)
	}

	var maxRequests int
	switch v := raw.(type) {
	case float64:
		maxRequests = int(v)
	case int:
		maxRequests = v
	default:
		return nil, fmt.Errorf("rate_limit rule %q: 'max_requests' must be a number, got %T", name, raw)
	}

	windowRaw, ok := config["window"]
	if !ok {
		return nil, fmt.Errorf("rate_limit rule %q: missing required config key 'window'", name)
	}
	windowStr, ok := windowRaw.(string)
	if !ok {
		return nil, fmt.Errorf("rate_limit rule %q: 'window' must be a string, got %T", name, windowRaw)
	}

	windowDur, err := parseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("rate_limit rule %q: invalid window %q: %w", name, windowStr, err)
	}

	perRaw, ok := config["per"]
	if !ok {
		return nil, fmt.Errorf("rate_limit rule %q: missing required config key 'per'", name)
	}
	per, ok := perRaw.(string)
	if !ok {
		return nil, fmt.Errorf("rate_limit rule %q: 'per' must be a string, got %T", name, perRaw)
	}

	switch per {
	case "global", "agent", "model":
	default:
		return nil, fmt.Errorf("rate_limit rule %q: 'per' must be one of global, agent, model; got %q", name, per)
	}

	return &RateLimitRule{
		name:        name,
		mode:        mode,
		maxRequests: maxRequests,
		windowDur:   windowDur,
		per:         per,
		counters:    make(map[string][]time.Time),
	}, nil
}

func (r *RateLimitRule) Name() string { return r.name }
func (r *RateLimitRule) Type() string { return "rate_limit" }
func (r *RateLimitRule) Mode() string { return r.mode }

// Evaluate checks if the current request exceeds the rate limit.
func (r *RateLimitRule) Evaluate(ctx policy.EvalContext) *policy.Violation {
	key := r.scopeKey(ctx)
	now := time.Now()
	cutoff := now.Add(-r.windowDur)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Prune old timestamps
	timestamps := r.counters[key]
	pruned := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}

	if len(pruned) >= r.maxRequests {
		count := len(pruned) + 1
		r.counters[key] = pruned
		return &policy.Violation{
			Rule: r.name,
			Type: r.Type(),
			Mode: r.mode,
			Message: fmt.Sprintf("Rate limit exceeded: %d/%d requests in %s (%s)",
				count, r.maxRequests, formatDuration(r.windowDur), r.per),
		}
	}

	// Record this request
	r.counters[key] = append(pruned, now)
	return nil
}

func (r *RateLimitRule) scopeKey(ctx policy.EvalContext) string {
	switch r.per {
	case "agent":
		return ctx.Agent
	case "model":
		return ctx.Model
	default:
		return "global"
	}
}

// formatDuration returns a compact string like "1m", "5m", "1h", "24h".
func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour && d%(24*time.Hour) == 0 {
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("%dd", days)
	}
	if d >= time.Hour && d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d >= time.Second && d%time.Second == 0 {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return d.String()
}

// parseDuration extends time.ParseDuration with support for "d" (days) suffix.
func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}
