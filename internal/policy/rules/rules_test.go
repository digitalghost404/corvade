package rules_test

import (
	"strings"
	"testing"

	"github.com/corvade/corvade/internal/policy"
	"github.com/corvade/corvade/internal/policy/rules"
)

// ---------------------------------------------------------------------------
// TokenLimitRule tests
// ---------------------------------------------------------------------------

func TestTokenLimitRule_UnderLimit(t *testing.T) {
	r, err := rules.NewTokenLimitRule("my-token-rule", "block", map[string]interface{}{
		"max_tokens": float64(10000),
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{TokenEstimate: 5000}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for under-limit request, got %+v", v)
	}
}

func TestTokenLimitRule_OverLimit(t *testing.T) {
	r, err := rules.NewTokenLimitRule("my-token-rule", "block", map[string]interface{}{
		"max_tokens": float64(10000),
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{TokenEstimate: 12450}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for over-limit request, got nil")
	}

	// Check fields
	if v.Rule != "my-token-rule" {
		t.Errorf("expected rule name %q, got %q", "my-token-rule", v.Rule)
	}
	if v.Type != "token_limit" {
		t.Errorf("expected type %q, got %q", "token_limit", v.Type)
	}
	if v.Mode != "block" {
		t.Errorf("expected mode %q, got %q", "block", v.Mode)
	}

	// Message should contain the token counts
	if !strings.Contains(v.Message, "12,450") {
		t.Errorf("expected message to contain %q, got %q", "12,450", v.Message)
	}
	if !strings.Contains(v.Message, "10,000") {
		t.Errorf("expected message to contain %q, got %q", "10,000", v.Message)
	}
}

func TestTokenLimitRule_ExactLimit(t *testing.T) {
	r, err := rules.NewTokenLimitRule("my-token-rule", "block", map[string]interface{}{
		"max_tokens": float64(10000),
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{TokenEstimate: 10000}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation at exact limit, got %+v", v)
	}
}

func TestTokenLimitRule_ConfigFloat64(t *testing.T) {
	r, err := rules.NewTokenLimitRule("r", "warn", map[string]interface{}{
		"max_tokens": float64(500),
	})
	if err != nil {
		t.Fatalf("unexpected error for float64 config: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil rule")
	}
}

func TestTokenLimitRule_ConfigInt(t *testing.T) {
	r, err := rules.NewTokenLimitRule("r", "warn", map[string]interface{}{
		"max_tokens": int(500),
	})
	if err != nil {
		t.Fatalf("unexpected error for int config: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil rule")
	}
}

func TestTokenLimitRule_MissingConfig(t *testing.T) {
	_, err := rules.NewTokenLimitRule("r", "warn", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing max_tokens, got nil")
	}
}

func TestTokenLimitRule_InvalidConfigType(t *testing.T) {
	_, err := rules.NewTokenLimitRule("r", "warn", map[string]interface{}{
		"max_tokens": "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for invalid max_tokens type, got nil")
	}
}

// ---------------------------------------------------------------------------
// ModelAllowlistRule tests
// ---------------------------------------------------------------------------

func TestModelAllowlistRule_ModelInAllowedList(t *testing.T) {
	r, err := rules.NewModelAllowlistRule("model-rule", "block", map[string]interface{}{
		"allowed": []interface{}{"gpt-4o-mini", "gpt-4o"},
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Model: "gpt-4o"}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for allowed model, got %+v", v)
	}
}

func TestModelAllowlistRule_ModelNotInAllowedList(t *testing.T) {
	r, err := rules.NewModelAllowlistRule("model-rule", "block", map[string]interface{}{
		"allowed": []interface{}{"gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Model: "gpt-4o"}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for model not in allowed list, got nil")
	}

	if v.Rule != "model-rule" {
		t.Errorf("expected rule name %q, got %q", "model-rule", v.Rule)
	}
	if v.Type != "model_allowlist" {
		t.Errorf("expected type %q, got %q", "model_allowlist", v.Type)
	}
	if v.Mode != "block" {
		t.Errorf("expected mode %q, got %q", "block", v.Mode)
	}
	if !strings.Contains(v.Message, "gpt-4o") {
		t.Errorf("expected message to mention model name, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "allowed") {
		t.Errorf("expected message to mention allowed list, got %q", v.Message)
	}
}

func TestModelAllowlistRule_ModelInBlockedList(t *testing.T) {
	r, err := rules.NewModelAllowlistRule("model-rule", "block", map[string]interface{}{
		"blocked": []interface{}{"gpt-4-turbo", "gpt-4"},
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Model: "gpt-4-turbo"}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for blocked model, got nil")
	}

	if !strings.Contains(v.Message, "gpt-4-turbo") {
		t.Errorf("expected message to mention model name, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "blocked") {
		t.Errorf("expected message to mention blocked list, got %q", v.Message)
	}
}

func TestModelAllowlistRule_ModelNotInBlockedList(t *testing.T) {
	r, err := rules.NewModelAllowlistRule("model-rule", "warn", map[string]interface{}{
		"blocked": []interface{}{"gpt-4-turbo"},
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Model: "gpt-4o-mini"}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for non-blocked model, got %+v", v)
	}
}

func TestModelAllowlistRule_BothAllowedAndBlocked(t *testing.T) {
	_, err := rules.NewModelAllowlistRule("model-rule", "block", map[string]interface{}{
		"allowed": []interface{}{"gpt-4o"},
		"blocked": []interface{}{"gpt-4-turbo"},
	})
	if err == nil {
		t.Fatal("expected error when both allowed and blocked are set, got nil")
	}
}

func TestModelAllowlistRule_NeitherSet(t *testing.T) {
	_, err := rules.NewModelAllowlistRule("model-rule", "block", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error when neither allowed nor blocked is set, got nil")
	}
}

func TestModelAllowlistRule_ConfigFromInterfaceSlice(t *testing.T) {
	// Simulate YAML unmarshalling which produces []interface{}
	r, err := rules.NewModelAllowlistRule("r", "warn", map[string]interface{}{
		"allowed": []interface{}{"claude-3-haiku", "claude-3-sonnet"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := policy.EvalContext{Model: "claude-3-haiku"}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for allowed model from interface{} slice, got %+v", v)
	}
}

func TestModelAllowlistRule_ImplementsRuleInterface(t *testing.T) {
	r, err := rules.NewModelAllowlistRule("r", "warn", map[string]interface{}{
		"allowed": []interface{}{"gpt-4o"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var _ policy.Rule = r
}

func TestTokenLimitRule_ImplementsRuleInterface(t *testing.T) {
	r, err := rules.NewTokenLimitRule("r", "warn", map[string]interface{}{
		"max_tokens": float64(1000),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var _ policy.Rule = r
}
