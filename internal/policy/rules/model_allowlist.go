package rules

import (
	"fmt"
	"strings"

	"github.com/corvade/corvade/internal/policy"
)

// ModelAllowlistRule enforces which models are permitted or forbidden.
type ModelAllowlistRule struct {
	name    string
	mode    string
	allowed []string
	blocked []string
}

// NewModelAllowlistRule constructs a ModelAllowlistRule from a raw config map.
// Exactly one of "allowed" or "blocked" must be set; setting both or neither is an error.
func NewModelAllowlistRule(name, mode string, config map[string]interface{}) (*ModelAllowlistRule, error) {
	allowed, hasAllowed, err := extractStringSlice(config, "allowed")
	if err != nil {
		return nil, fmt.Errorf("model_allowlist rule %q: 'allowed': %w", name, err)
	}

	blocked, hasBlocked, err := extractStringSlice(config, "blocked")
	if err != nil {
		return nil, fmt.Errorf("model_allowlist rule %q: 'blocked': %w", name, err)
	}

	if hasAllowed && hasBlocked {
		return nil, fmt.Errorf("model_allowlist rule %q: only one of 'allowed' or 'blocked' may be set, not both", name)
	}
	if !hasAllowed && !hasBlocked {
		return nil, fmt.Errorf("model_allowlist rule %q: one of 'allowed' or 'blocked' must be set", name)
	}

	return &ModelAllowlistRule{
		name:    name,
		mode:    mode,
		allowed: allowed,
		blocked: blocked,
	}, nil
}

func (r *ModelAllowlistRule) Name() string { return r.name }
func (r *ModelAllowlistRule) Type() string { return "model_allowlist" }
func (r *ModelAllowlistRule) Mode() string { return r.mode }

// Evaluate returns a Violation when the model in ctx violates the allowlist/blocklist.
func (r *ModelAllowlistRule) Evaluate(ctx policy.EvalContext) *policy.Violation {
	if len(r.allowed) > 0 {
		if !contains(r.allowed, ctx.Model) {
			return &policy.Violation{
				Rule:    r.name,
				Type:    r.Type(),
				Mode:    r.mode,
				Message: fmt.Sprintf("Model %s is not in the allowed list [%s]", ctx.Model, strings.Join(r.allowed, ", ")),
			}
		}
		return nil
	}

	// blocked list
	if contains(r.blocked, ctx.Model) {
		return &policy.Violation{
			Rule:    r.name,
			Type:    r.Type(),
			Mode:    r.mode,
			Message: fmt.Sprintf("Model %s is in the blocked list", ctx.Model),
		}
	}
	return nil
}

// extractStringSlice pulls a []string from config[key] where the value may be
// a []interface{} (from YAML unmarshalling) or a []string (from direct Go usage).
// Returns the slice, a boolean indicating whether the key was present, and any error.
func extractStringSlice(config map[string]interface{}, key string) ([]string, bool, error) {
	raw, ok := config[key]
	if !ok {
		return nil, false, nil
	}

	switch v := raw.(type) {
	case []string:
		return v, true, nil
	case []interface{}:
		result := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, true, fmt.Errorf("element %d must be a string, got %T", i, item)
			}
			result = append(result, s)
		}
		return result, true, nil
	default:
		return nil, true, fmt.Errorf("must be a list of strings, got %T", raw)
	}
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
