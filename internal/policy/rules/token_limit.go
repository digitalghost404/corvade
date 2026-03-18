package rules

import (
	"fmt"
	"strconv"

	"github.com/corvade/corvade/internal/policy"
)

// TokenLimitRule blocks or warns when the estimated token count exceeds a maximum.
type TokenLimitRule struct {
	name      string
	mode      string
	maxTokens int
}

// NewTokenLimitRule constructs a TokenLimitRule from a raw config map.
// The config must contain "max_tokens" as either float64 (YAML default) or int.
func NewTokenLimitRule(name, mode string, config map[string]interface{}) (*TokenLimitRule, error) {
	raw, ok := config["max_tokens"]
	if !ok {
		return nil, fmt.Errorf("token_limit rule %q: missing required config key 'max_tokens'", name)
	}

	var maxTokens int
	switch v := raw.(type) {
	case float64:
		maxTokens = int(v)
	case int:
		maxTokens = v
	default:
		return nil, fmt.Errorf("token_limit rule %q: 'max_tokens' must be a number, got %T", name, raw)
	}

	return &TokenLimitRule{
		name:      name,
		mode:      mode,
		maxTokens: maxTokens,
	}, nil
}

func (r *TokenLimitRule) Name() string { return r.name }
func (r *TokenLimitRule) Type() string { return "token_limit" }
func (r *TokenLimitRule) Mode() string { return r.mode }

// Evaluate returns a Violation if ctx.TokenEstimate exceeds the configured maximum.
func (r *TokenLimitRule) Evaluate(ctx policy.EvalContext) *policy.Violation {
	if ctx.TokenEstimate <= r.maxTokens {
		return nil
	}

	msg := fmt.Sprintf(
		"Estimated %s tokens exceeds limit of %s",
		commaInt(ctx.TokenEstimate),
		commaInt(r.maxTokens),
	)

	return &policy.Violation{
		Rule:    r.name,
		Type:    r.Type(),
		Mode:    r.mode,
		Message: msg,
	}
}

// commaInt formats an integer with thousands-separator commas (e.g. 12450 → "12,450").
func commaInt(n int) string {
	s := strconv.Itoa(n)
	// Insert commas from right to left every three digits.
	out := make([]byte, 0, len(s)+len(s)/3)
	start := len(s) % 3
	if start == 0 {
		start = 3
	}
	out = append(out, s[:start]...)
	for i := start; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}
