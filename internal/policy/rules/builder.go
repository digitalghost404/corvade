package rules

import (
	"fmt"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/policy"
)

// DefaultBuilder returns a RuleBuilderFunc that constructs all built-in rule types.
func DefaultBuilder() policy.RuleBuilderFunc {
	return func(rc policy.RuleConfig, store *capture.Store) (policy.Rule, error) {
		switch rc.Type {
		case "token_limit":
			return NewTokenLimitRule(rc.Name, rc.Mode, rc.Config)
		case "model_allowlist":
			return NewModelAllowlistRule(rc.Name, rc.Mode, rc.Config)
		case "rate_limit":
			return NewRateLimitRule(rc.Name, rc.Mode, rc.Config)
		case "pii_redaction":
			return NewPIIRedactionRule(rc.Name, rc.Mode, rc.Config)
		case "cost_ceiling":
			return NewCostCeilingRule(rc.Name, rc.Mode, rc.Config, store)
		default:
			return nil, fmt.Errorf("rule %q: unknown type %q", rc.Name, rc.Type)
		}
	}
}
