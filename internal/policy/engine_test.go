package policy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/policy"
	"github.com/corvade/corvade/internal/policy/rules"
)

func writePolicyFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
	return path
}

func newTestEngine(t *testing.T, path string) *policy.Engine {
	t.Helper()
	eng, err := policy.NewEngine(path, nil, nil, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return eng
}

const validPolicy = `
rules:
  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 8000
  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
        - claude-sonnet-4-20250514
`

const observeOnlyPolicy = `
rules:
  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`

const enforcePolicy = `
rules:
  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
`

const mixedPolicy = `
rules:
  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
  - name: token-cap-2
    type: token_limit
    mode: observe
    config:
      max_tokens: 50
`

const piiRedactPolicy = `
rules:
  - name: pii-filter
    type: pii_redaction
    mode: enforce
    config:
      action: redact
`

const invalidYAML = `
rules:
  - name: [broken
`

const invalidModePolicy = `
rules:
  - name: bad-mode
    type: token_limit
    mode: yolo
    config:
      max_tokens: 100
`

const invalidRuleTypePolicy = `
rules:
  - name: mystery
    type: does_not_exist
    mode: observe
    config: {}
`

func TestNewEngine_ValidPolicy(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, validPolicy)

	eng := newTestEngine(t, path)

	rs := eng.Rules()
	if len(rs) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rs))
	}

	if rs[0].Name != "token-cap" || rs[0].Type != "token_limit" || rs[0].Mode != "observe" {
		t.Errorf("unexpected rule 0: %+v", rs[0])
	}
	if rs[1].Name != "model-gate" || rs[1].Type != "model_allowlist" || rs[1].Mode != "enforce" {
		t.Errorf("unexpected rule 1: %+v", rs[1])
	}
	for _, r := range rs {
		if !r.Valid {
			t.Errorf("rule %q should be valid", r.Name)
		}
	}
}

func TestNewEngine_MissingFile(t *testing.T) {
	eng := newTestEngine(t, "/tmp/does-not-exist-policy.yaml")
	if len(eng.Rules()) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(eng.Rules()))
	}
}

func TestNewEngine_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidYAML)

	_, err := policy.NewEngine(path, nil, nil, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestNewEngine_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidModePolicy)

	_, err := policy.NewEngine(path, nil, nil, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
}

func TestNewEngine_UnknownRuleType(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidRuleTypePolicy)

	_, err := policy.NewEngine(path, nil, nil, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err == nil {
		t.Fatal("expected error for unknown rule type, got nil")
	}
}

func TestEvaluate_NoRules(t *testing.T) {
	eng := newTestEngine(t, "/tmp/does-not-exist-policy.yaml")

	result := eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 1000,
	})

	if result.Blocked {
		t.Error("expected not blocked with no rules")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
}

func TestEvaluate_ObserveModeViolation(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, observeOnlyPolicy)
	eng := newTestEngine(t, path)

	result := eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 5000, // exceeds 100
	})

	if result.Blocked {
		t.Error("observe mode should not block")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Mode != "observe" {
		t.Errorf("violation mode should be observe, got %q", result.Violations[0].Mode)
	}
}

func TestEvaluate_EnforceModeViolation(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, enforcePolicy)
	eng := newTestEngine(t, path)

	result := eng.Evaluate(policy.EvalContext{
		Model:         "claude-sonnet-4-20250514", // not in allowed list
		TokenEstimate: 100,
	})

	if !result.Blocked {
		t.Error("enforce mode should block")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
}

func TestEvaluate_MixedObserveEnforce(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, mixedPolicy)
	eng := newTestEngine(t, path)

	// Token exceeds both observe limits (100, 50), model not in allowed list (enforce)
	result := eng.Evaluate(policy.EvalContext{
		Model:         "claude-sonnet-4-20250514",
		TokenEstimate: 5000,
	})

	if !result.Blocked {
		t.Error("should be blocked by enforce rule")
	}

	// token-cap (observe) fires, then model-gate (enforce) fires and stops.
	// token-cap-2 should NOT be evaluated because enforce already blocked.
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations (1 observe + 1 enforce), got %d: %+v", len(result.Violations), result.Violations)
	}

	if result.Violations[0].Mode != "observe" {
		t.Errorf("first violation should be observe, got %q", result.Violations[0].Mode)
	}
	if result.Violations[1].Mode != "enforce" {
		t.Errorf("second violation should be enforce, got %q", result.Violations[1].Mode)
	}
}

func TestEvaluate_PIIRedaction(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, piiRedactPolicy)
	eng := newTestEngine(t, path)

	body, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "My email is test@example.com"},
		},
	})

	result := eng.Evaluate(policy.EvalContext{
		Model:       "gpt-4o",
		RequestBody: body,
	})

	if len(result.Violations) == 0 {
		t.Fatal("expected PII violation")
	}
	if result.Modified == nil {
		t.Fatal("expected modified body from PII redaction")
	}
	if string(result.Modified) == string(body) {
		t.Error("modified body should differ from original")
	}
}

func TestHotReload_UpdatesRules(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, validPolicy)
	eng := newTestEngine(t, path)

	if len(eng.Rules()) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(eng.Rules()))
	}

	eng.Start()
	defer eng.Stop()

	// Overwrite with a single-rule policy
	time.Sleep(100 * time.Millisecond) // small gap so mtime changes
	writePolicyFile(t, dir, observeOnlyPolicy)

	// Wait for the watcher to pick up the change (poll every 2s)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(eng.Rules()) == 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if len(eng.Rules()) != 1 {
		t.Fatalf("expected 1 rule after hot reload, got %d", len(eng.Rules()))
	}
	if eng.Rules()[0].Name != "token-cap" {
		t.Errorf("expected rule name token-cap, got %q", eng.Rules()[0].Name)
	}
}

func TestHotReload_InvalidFileKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, validPolicy)
	eng := newTestEngine(t, path)

	if len(eng.Rules()) != 2 {
		t.Fatalf("expected 2 rules initially, got %d", len(eng.Rules()))
	}

	eng.Start()
	defer eng.Stop()

	// Overwrite with invalid YAML
	time.Sleep(100 * time.Millisecond)
	writePolicyFile(t, dir, invalidYAML)

	// Wait long enough for the watcher to attempt reload
	time.Sleep(3 * time.Second)

	// Should still have the previous 2 rules
	if len(eng.Rules()) != 2 {
		t.Fatalf("expected 2 rules after invalid reload, got %d", len(eng.Rules()))
	}
}

func TestRules_ReturnsSummaries(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, validPolicy)
	eng := newTestEngine(t, path)

	summaries := eng.Rules()
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	for _, s := range summaries {
		if s.Name == "" || s.Type == "" || s.Mode == "" {
			t.Errorf("incomplete summary: %+v", s)
		}
		if !s.Valid {
			t.Errorf("rule %q should be valid", s.Name)
		}
	}
}

func TestValidate_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, validPolicy)

	summaries, err := policy.Validate(path, rules.DefaultBuilder())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
}

func TestValidate_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidYAML)

	_, err := policy.Validate(path, rules.DefaultBuilder())
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestValidate_MissingFile(t *testing.T) {
	_, err := policy.Validate("/tmp/does-not-exist-validate.yaml", rules.DefaultBuilder())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestNewEngine_NoBuilder(t *testing.T) {
	_, err := policy.NewEngine("/tmp/does-not-exist.yaml", nil, nil)
	if err == nil {
		t.Fatal("expected error when no builder is provided")
	}
}

func TestValidate_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidModePolicy)

	_, err := policy.Validate(path, rules.DefaultBuilder())
	if err == nil {
		t.Fatal("expected error for invalid mode in Validate")
	}
}

func TestValidate_UnknownRuleType(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, invalidRuleTypePolicy)

	_, err := policy.Validate(path, rules.DefaultBuilder())
	if err == nil {
		t.Fatal("expected error for unknown rule type in Validate")
	}
}

func TestEvaluate_EnforceNoViolation(t *testing.T) {
	dir := t.TempDir()
	path := writePolicyFile(t, dir, enforcePolicy)
	eng := newTestEngine(t, path)

	// Model IS in allowed list — no violation
	result := eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 100,
	})

	if result.Blocked {
		t.Error("should not be blocked when model is allowed")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
}
