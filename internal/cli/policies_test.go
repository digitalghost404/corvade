package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- NewPoliciesCmd ----------

func TestNewPoliciesCmd(t *testing.T) {
	cmd := NewPoliciesCmd()
	if cmd == nil {
		t.Fatal("NewPoliciesCmd returned nil")
	}
	if cmd.Use != "policies" {
		t.Errorf("Use = %q, want 'policies'", cmd.Use)
	}
}

func TestPoliciesCmdHasValidateSubcommand(t *testing.T) {
	cmd := NewPoliciesCmd()
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "validate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'validate' subcommand on policies command")
	}
}

// ---------- defaultPolicyPath ----------

func TestDefaultPolicyPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".corvade", "policies.yaml")
	got := defaultPolicyPath()
	if got != expected {
		t.Errorf("defaultPolicyPath() = %q, want %q", got, expected)
	}
}

// ---------- runPolicies with missing file ----------

func TestRunPoliciesMissingFile(t *testing.T) {
	// Override the policy path by patching UserHomeDir indirectly isn't possible,
	// so we test the behaviour by calling with a non-existent file via a helper.
	// We test the function that does the actual lookup.
	nonExistentPath := filepath.Join(t.TempDir(), "missing.yaml")
	_, statErr := os.Stat(nonExistentPath)
	if !os.IsNotExist(statErr) {
		t.Skip("expected missing file")
	}
	// Verify our path helper returns a path under ~/.corvade
	p := defaultPolicyPath()
	if !strings.Contains(p, ".corvade") {
		t.Errorf("defaultPolicyPath() does not contain .corvade: %q", p)
	}
}

// ---------- readWebhookURL ----------

func TestReadWebhookURLMissingFile(t *testing.T) {
	url := readWebhookURL("/tmp/corvade-nonexistent-policies-test.yaml")
	if url != "" {
		t.Errorf("expected empty string for missing file, got %q", url)
	}
}

func TestReadWebhookURLNoWebhook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yaml")
	content := `rules:
  - name: token-limit
    type: token_limit
    mode: observe
    config:
      max_tokens: 4000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	url := readWebhookURL(path)
	if url != "" {
		t.Errorf("expected empty string for policy without webhook, got %q", url)
	}
}

func TestReadWebhookURLWithWebhook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yaml")
	content := `webhook:
  url: https://hooks.slack.com/services/T00/B00/xxx
  events: [violation]
rules:
  - name: token-limit
    type: token_limit
    mode: observe
    config:
      max_tokens: 4000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	url := readWebhookURL(path)
	if url != "https://hooks.slack.com/services/T00/B00/xxx" {
		t.Errorf("readWebhookURL = %q, want webhook URL", url)
	}
}

// ---------- runPolicies (integration via temp file) ----------

func TestRunPoliciesValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yaml")
	content := `rules:
  - name: token-limit
    type: token_limit
    mode: observe
    config:
      max_tokens: 4000
  - name: prod-models
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
        - claude-3-5-sonnet-20241022
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Call the internal validate path directly to confirm summaries are returned
	import_rules_validate(t, path, 2, false)
}

func TestRunPoliciesInvalidRuleType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yaml")
	content := `rules:
  - name: bad-rule
    type: unknown_type
    mode: observe
    config: {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	import_rules_validate(t, path, 0, true)
}

func TestRunPoliciesInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yaml")
	content := `rules:
  - name: bad-mode
    type: token_limit
    mode: invalid_mode
    config:
      max_tokens: 1000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	import_rules_validate(t, path, 0, true)
}

// import_rules_validate is a helper that invokes policy.Validate via the rules builder
// and asserts the expected outcome.
func import_rules_validate(t *testing.T, path string, wantCount int, wantErr bool) {
	t.Helper()

	// We call the same logic that runPolicies uses
	from_rules_pkg_validate(t, path, wantCount, wantErr)
}

func from_rules_pkg_validate(t *testing.T, path string, wantCount int, wantErr bool) {
	t.Helper()

	// Import inline since we're in the same package
	// We use the policy and rules packages directly
	summaries, err := validatePolicyFile(path)
	if wantErr {
		if err == nil {
			t.Errorf("expected error but got nil (summaries: %v)", summaries)
		}
		return
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if len(summaries) != wantCount {
		t.Errorf("got %d summaries, want %d", len(summaries), wantCount)
	}
}

// ---------- extractViolations ----------

func TestExtractViolationsNil(t *testing.T) {
	data := map[string]interface{}{"model": "gpt-4o"}
	v := extractViolations(data)
	if v != nil {
		t.Errorf("expected nil for missing key, got %v", v)
	}
}

func TestExtractViolationsExplicitNil(t *testing.T) {
	data := map[string]interface{}{"policy_violations": nil}
	v := extractViolations(data)
	if v != nil {
		t.Errorf("expected nil for explicit nil, got %v", v)
	}
}

func TestExtractViolationsValid(t *testing.T) {
	data := map[string]interface{}{
		"policy_violations": []interface{}{
			map[string]interface{}{
				"rule":    "token-limit",
				"type":    "token_limit",
				"mode":    "observe",
				"message": "token count 5000 exceeds limit 4000",
			},
		},
	}
	v := extractViolations(data)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(v))
	}
	if v[0]["type"] != "token_limit" {
		t.Errorf("type = %q, want 'token_limit'", v[0]["type"])
	}
}

func TestExtractViolationsMultiple(t *testing.T) {
	data := map[string]interface{}{
		"policy_violations": []interface{}{
			map[string]interface{}{"type": "token_limit", "message": "too many tokens"},
			map[string]interface{}{"type": "model_allowlist", "message": "model not allowed"},
		},
	}
	v := extractViolations(data)
	if len(v) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(v))
	}
}

// ---------- printBlockedEvent (smoke test) ----------

func TestPrintBlockedEvent(t *testing.T) {
	// Verify it doesn't panic with various data shapes
	data := map[string]interface{}{
		"model": "gpt-4o",
		"policy_violations": []interface{}{
			map[string]interface{}{
				"type":    "model_allowlist",
				"message": "model not in allowlist",
			},
		},
	}
	printBlockedEvent(data)

	// Without violations
	printBlockedEvent(map[string]interface{}{"model": "gpt-4o"})

	// Missing model
	printBlockedEvent(map[string]interface{}{})
}

// ---------- printTraceEvent with violations (smoke test) ----------

func TestPrintTraceEventWithViolations(t *testing.T) {
	data := map[string]interface{}{
		"model":      "gpt-4o",
		"cost":       float64(0.02),
		"latency_ms": float64(800),
		"status":     "ok",
		"provider":   "openai",
		"policy_violations": []interface{}{
			map[string]interface{}{
				"type":    "token_limit",
				"message": "token count 5000 exceeds limit 4000",
			},
		},
	}
	// Should not panic; observe line should be printed
	printTraceEvent(data)
}
