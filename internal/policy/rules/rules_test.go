package rules_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
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

// ---------------------------------------------------------------------------
// RateLimitRule tests
// ---------------------------------------------------------------------------

func TestRateLimitRule_FirstRequest_Passes(t *testing.T) {
	r, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(10),
		"window":       "1m",
		"per":          "global",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Agent: "bot", Model: "gpt-4o"}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for first request, got %+v", v)
	}
}

func TestRateLimitRule_ExceedsLimit(t *testing.T) {
	r, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(5),
		"window":       "1m",
		"per":          "global",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Agent: "bot", Model: "gpt-4o"}
	// Use up the limit
	for i := 0; i < 5; i++ {
		v := r.Evaluate(ctx)
		if v != nil {
			t.Fatalf("expected nil violation on request %d, got %+v", i+1, v)
		}
	}

	// 6th request should violate
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation on request exceeding limit, got nil")
	}
	if v.Type != "rate_limit" {
		t.Errorf("expected type %q, got %q", "rate_limit", v.Type)
	}
	if !strings.Contains(v.Message, "6/5") {
		t.Errorf("expected message to contain '6/5', got %q", v.Message)
	}
	if !strings.Contains(v.Message, "global") {
		t.Errorf("expected message to contain 'global', got %q", v.Message)
	}
}

func TestRateLimitRule_WindowExpiry(t *testing.T) {
	r, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(2),
		"window":       "100ms",
		"per":          "global",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{}
	// Fill the window
	r.Evaluate(ctx)
	r.Evaluate(ctx)

	// Should be at limit
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation at limit, got nil")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should pass again
	v = r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation after window expiry, got %+v", v)
	}
}

func TestRateLimitRule_PerAgent_IndependentCounters(t *testing.T) {
	r, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(2),
		"window":       "1m",
		"per":          "agent",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctxA := policy.EvalContext{Agent: "agent-a"}
	ctxB := policy.EvalContext{Agent: "agent-b"}

	// Fill agent-a's limit
	r.Evaluate(ctxA)
	r.Evaluate(ctxA)

	// agent-a should be blocked
	v := r.Evaluate(ctxA)
	if v == nil {
		t.Fatal("expected violation for agent-a at limit, got nil")
	}

	// agent-b should still be allowed
	v = r.Evaluate(ctxB)
	if v != nil {
		t.Errorf("expected nil violation for agent-b, got %+v", v)
	}
}

func TestRateLimitRule_InvalidConfig_MissingFields(t *testing.T) {
	_, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing config fields, got nil")
	}

	_, err = rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(10),
	})
	if err == nil {
		t.Fatal("expected error for missing window, got nil")
	}

	_, err = rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(10),
		"window":       "1m",
	})
	if err == nil {
		t.Fatal("expected error for missing per, got nil")
	}
}

func TestRateLimitRule_ImplementsRuleInterface(t *testing.T) {
	r, err := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(10),
		"window":       "1m",
		"per":          "global",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var _ policy.Rule = r
}

// ---------------------------------------------------------------------------
// PIIRedactionRule tests
// ---------------------------------------------------------------------------

func TestPIIRedactionRule_NoPII(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"Hello, how are you?"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for no PII, got %+v", v)
	}
}

func TestPIIRedactionRule_EmailFound(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"Contact me at test@example.com and admin@corp.co"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for email PII, got nil")
	}
	if !strings.Contains(v.Message, "email") {
		t.Errorf("expected message to mention email, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "2") {
		t.Errorf("expected message to mention count of 2, got %q", v.Message)
	}
}

func TestPIIRedactionRule_MultiplePatterns(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"Email: test@example.com, phone: 555-123-4567"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for multiple PII types, got nil")
	}
	if !strings.Contains(v.Message, "email") {
		t.Errorf("expected message to mention email, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "phone") {
		t.Errorf("expected message to mention phone, got %q", v.Message)
	}
}

func TestPIIRedactionRule_ActionBlock_NoModified(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "enforce", map[string]interface{}{
		"action": "block",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"test@example.com"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	result := r.EvaluateWithResult(ctx)
	if result == nil {
		t.Fatal("expected result for blocked PII, got nil")
	}
	if result.Modified != nil {
		t.Errorf("expected nil Modified for block action, got %s", string(result.Modified))
	}
}

func TestPIIRedactionRule_ActionRedact_ModifiedBody(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "enforce", map[string]interface{}{
		"action": "redact",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"Contact test@example.com please"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	result := r.EvaluateWithResult(ctx)
	if result == nil {
		t.Fatal("expected result for redacted PII, got nil")
	}
	if result.Modified == nil {
		t.Fatal("expected Modified body for redact action, got nil")
	}
	if !strings.Contains(string(result.Modified), "[REDACTED]") {
		t.Errorf("expected Modified body to contain [REDACTED], got %s", string(result.Modified))
	}
	if strings.Contains(string(result.Modified), "test@example.com") {
		t.Errorf("expected email to be redacted, but still present in Modified body")
	}
	// Verify it's still valid JSON
	if !json.Valid(result.Modified) {
		t.Errorf("expected Modified body to be valid JSON, got %s", string(result.Modified))
	}
}

func TestPIIRedactionRule_MalformedBody(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{RequestBody: []byte("not json at all")}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for malformed body, got %+v", v)
	}
}

func TestPIIRedactionRule_EmptyBody(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{RequestBody: nil}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for empty body, got %+v", v)
	}
}

func TestPIIRedactionRule_SSNDetection(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"My SSN is 123-45-6789"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for SSN, got nil")
	}
	if !strings.Contains(v.Message, "ssn") {
		t.Errorf("expected message to mention ssn, got %q", v.Message)
	}
}

func TestPIIRedactionRule_CreditCardDetection(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"Card: 4111-1111-1111-1111"}]}`
	ctx := policy.EvalContext{RequestBody: []byte(body)}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for credit card, got nil")
	}
	if !strings.Contains(v.Message, "credit_card") {
		t.Errorf("expected message to mention credit_card, got %q", v.Message)
	}
}

func TestPIIRedactionRule_ImplementsRuleInterface(t *testing.T) {
	r, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var _ policy.Rule = r
}

// ---------------------------------------------------------------------------
// CostCeilingRule tests
// ---------------------------------------------------------------------------

func newTestStore(t *testing.T) *capture.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := capture.NewStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func insertTestTrace(t *testing.T, store *capture.Store, agent string, cost float64) {
	t.Helper()
	agentPtr := &agent
	costPtr := &cost
	_, err := store.InsertTrace(capture.Trace{
		Provider:   "openai",
		Model:      "gpt-4o",
		Request:    "{}",
		StatusCode: 200,
		Agent:      agentPtr,
		Cost:       costPtr,
	})
	if err != nil {
		t.Fatalf("failed to insert test trace: %v", err)
	}
}

func TestCostCeilingRule_UnderBudget(t *testing.T) {
	store := newTestStore(t)
	insertTestTrace(t, store, "bot", 1.50)

	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 5.0,
		"window":   "24h",
		"per":      "global",
	}, store)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Agent: "bot", EstimatedCost: 0.50}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation under budget, got %+v", v)
	}
}

func TestCostCeilingRule_OverBudget(t *testing.T) {
	store := newTestStore(t)
	insertTestTrace(t, store, "bot", 4.50)

	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 5.0,
		"window":   "24h",
		"per":      "global",
	}, store)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Agent: "bot", EstimatedCost: 1.00}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation over budget, got nil")
	}
	if !strings.Contains(v.Message, "Cost ceiling exceeded") {
		t.Errorf("expected cost ceiling message, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "$4.50") {
		t.Errorf("expected message to contain spent amount, got %q", v.Message)
	}
	if !strings.Contains(v.Message, "$5.00") {
		t.Errorf("expected message to contain limit, got %q", v.Message)
	}
}

func TestCostCeilingRule_CacheWorks(t *testing.T) {
	store := newTestStore(t)
	insertTestTrace(t, store, "bot", 1.00)

	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 100.0,
		"window":   "24h",
		"per":      "global",
	}, store)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := policy.EvalContext{Agent: "bot", EstimatedCost: 0.10}

	// First call populates cache
	r.Evaluate(ctx)

	// Insert more cost - but cache should prevent seeing it immediately
	insertTestTrace(t, store, "bot", 50.00)

	// Second rapid call should use cached value (still $1.00, not $51.00)
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation (cached value should be $1.00), got %+v", v)
	}
}

func TestCostCeilingRule_PerAgent_DifferentScopes(t *testing.T) {
	store := newTestStore(t)
	insertTestTrace(t, store, "expensive-bot", 9.00)
	insertTestTrace(t, store, "cheap-bot", 1.00)

	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 10.0,
		"window":   "24h",
		"per":      "agent",
	}, store)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	// expensive-bot should be over budget
	ctxExpensive := policy.EvalContext{Agent: "expensive-bot", EstimatedCost: 2.00}
	v := r.Evaluate(ctxExpensive)
	if v == nil {
		t.Fatal("expected violation for expensive-bot, got nil")
	}

	// cheap-bot should be fine
	ctxCheap := policy.EvalContext{Agent: "cheap-bot", EstimatedCost: 2.00}
	v = r.Evaluate(ctxCheap)
	if v != nil {
		t.Errorf("expected nil violation for cheap-bot, got %+v", v)
	}
}

func TestCostCeilingRule_MissingStore(t *testing.T) {
	_, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 5.0,
		"window":   "24h",
		"per":      "global",
	}, nil)
	if err == nil {
		t.Fatal("expected error for nil store, got nil")
	}
}

func TestCostCeilingRule_MissingConfig(t *testing.T) {
	store := newTestStore(t)

	_, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{}, store)
	if err == nil {
		t.Fatal("expected error for missing max_cost, got nil")
	}
}

func TestCostCeilingRule_WindowDays(t *testing.T) {
	store := newTestStore(t)

	// 7d window should parse correctly
	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 100.0,
		"window":   "7d",
		"per":      "global",
	}, store)
	if err != nil {
		t.Fatalf("unexpected error for 7d window: %v", err)
	}

	ctx := policy.EvalContext{EstimatedCost: 1.00}
	v := r.Evaluate(ctx)
	if v != nil {
		t.Errorf("expected nil violation for empty store, got %+v", v)
	}
}

func TestCostCeilingRule_ImplementsRuleInterface(t *testing.T) {
	store := newTestStore(t)
	r, err := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 5.0,
		"window":   "24h",
		"per":      "global",
	}, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var _ policy.Rule = r
}

// ---------------------------------------------------------------------------
// Additional coverage tests for Name()/Mode() accessors and edge cases
// ---------------------------------------------------------------------------

func TestRateLimitRule_NameAndMode(t *testing.T) {
	r, _ := rules.NewRateLimitRule("my-rl", "warn", map[string]interface{}{
		"max_requests": float64(10), "window": "1m", "per": "global",
	})
	if r.Name() != "my-rl" {
		t.Errorf("expected name %q, got %q", "my-rl", r.Name())
	}
	if r.Mode() != "warn" {
		t.Errorf("expected mode %q, got %q", "warn", r.Mode())
	}
}

func TestRateLimitRule_PerModel(t *testing.T) {
	r, _ := rules.NewRateLimitRule("rl", "block", map[string]interface{}{
		"max_requests": float64(1), "window": "1m", "per": "model",
	})

	ctxA := policy.EvalContext{Model: "gpt-4o"}
	ctxB := policy.EvalContext{Model: "claude-3"}

	r.Evaluate(ctxA) // fill gpt-4o
	v := r.Evaluate(ctxA)
	if v == nil {
		t.Fatal("expected violation for gpt-4o at limit")
	}
	v = r.Evaluate(ctxB)
	if v != nil {
		t.Errorf("expected nil for claude-3, got %+v", v)
	}
}

func TestPIIRedactionRule_NameAndMode(t *testing.T) {
	r, _ := rules.NewPIIRedactionRule("my-pii", "warn", map[string]interface{}{})
	if r.Name() != "my-pii" {
		t.Errorf("expected name %q, got %q", "my-pii", r.Name())
	}
	if r.Mode() != "warn" {
		t.Errorf("expected mode %q, got %q", "warn", r.Mode())
	}
}

func TestCostCeilingRule_NameAndMode(t *testing.T) {
	store := newTestStore(t)
	r, _ := rules.NewCostCeilingRule("my-cost", "warn", map[string]interface{}{
		"max_cost": 10.0, "window": "1h", "per": "global",
	}, store)
	if r.Name() != "my-cost" {
		t.Errorf("expected name %q, got %q", "my-cost", r.Name())
	}
	if r.Mode() != "warn" {
		t.Errorf("expected mode %q, got %q", "warn", r.Mode())
	}
}

func TestCostCeilingRule_PerSession(t *testing.T) {
	store := newTestStore(t)
	// Insert a trace with a session
	sessID := "sess-123"
	cost := 3.0
	_, err := store.InsertTrace(capture.Trace{
		Provider:   "openai",
		Model:      "gpt-4o",
		Request:    "{}",
		StatusCode: 200,
		SessionID:  &sessID,
		Cost:       &cost,
	})
	if err != nil {
		t.Fatalf("insert trace: %v", err)
	}

	r, _ := rules.NewCostCeilingRule("cost", "block", map[string]interface{}{
		"max_cost": 5.0, "window": "24h", "per": "session",
	}, store)

	ctx := policy.EvalContext{Session: "sess-123", EstimatedCost: 3.0}
	v := r.Evaluate(ctx)
	if v == nil {
		t.Fatal("expected violation for session over budget")
	}

	// Different session should pass
	ctx2 := policy.EvalContext{Session: "sess-other", EstimatedCost: 1.0}
	v = r.Evaluate(ctx2)
	if v != nil {
		t.Errorf("expected nil for different session, got %+v", v)
	}
}

func TestPIIRedactionRule_InvalidAction(t *testing.T) {
	_, err := rules.NewPIIRedactionRule("pii", "block", map[string]interface{}{
		"action": "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid action, got nil")
	}
}

// Ensure unused imports are used.
var _ = os.TempDir
var _ = time.Now
