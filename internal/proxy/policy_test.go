package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/policy"
	"github.com/corvade/corvade/internal/policy/rules"
)

// newTestPolicyEngine creates a policy engine from a YAML string written to a temp file.
func newTestPolicyEngine(t *testing.T, yamlContent string, store *capture.Store) *policy.Engine {
	t.Helper()
	dir := t.TempDir()
	policyPath := dir + "/policy.yaml"

	if err := os.WriteFile(policyPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	calc := cost.NewCalculator(nil)
	engine, err := policy.NewEngine(policyPath, store, calc, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err != nil {
		t.Fatalf("failed to create policy engine: %v", err)
	}
	return engine
}

// --- Test 1: Policy blocks request (enforce token_limit) ---

func TestPolicyBlocksRequest(t *testing.T) {
	upstream := httptest.NewServer(mockOpenAIUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	yamlPolicy := `
rules:
  - name: tiny-limit
    type: token_limit
    mode: enforce
    config:
      max_tokens: 1
`
	engine := newTestPolicyEngine(t, yamlPolicy, store)
	srv.SetPolicyEngine(engine)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello world this is a test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Verify: status 400
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify: X-Corvade-Policy-Violation header present
	if rec.Header().Get("X-Corvade-Policy-Violation") != "true" {
		t.Error("expected X-Corvade-Policy-Violation header to be 'true'")
	}

	// Verify: OpenAI error format
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Type != "policy_violation" {
		t.Errorf("expected error type 'policy_violation', got %q", errResp.Error.Type)
	}
	if errResp.Error.Code != "corvade_policy_blocked" {
		t.Errorf("expected error code 'corvade_policy_blocked', got %q", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "Corvade policy") {
		t.Errorf("expected message to contain 'Corvade policy', got %q", errResp.Error.Message)
	}

	// Verify: trace captured with status 499
	time.Sleep(300 * time.Millisecond)
	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].StatusCode != 499 {
		t.Errorf("expected trace status 499, got %d", traces[0].StatusCode)
	}
	if traces[0].PolicyViolations == nil {
		t.Error("expected trace to have policy violations")
	}
}

// --- Test 2: Policy observes but doesn't block ---

func TestPolicyObserveMode(t *testing.T) {
	upstream := httptest.NewServer(mockOpenAIUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	yamlPolicy := `
rules:
  - name: soft-limit
    type: token_limit
    mode: observe
    config:
      max_tokens: 1
`
	engine := newTestPolicyEngine(t, yamlPolicy, store)
	srv.SetPolicyEngine(engine)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello world this is a test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Verify: request forwarded (200)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "chatcmpl-test123") {
		t.Fatalf("expected response from upstream, got: %s", body)
	}

	// Verify: trace has violations attached
	time.Sleep(300 * time.Millisecond)
	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].PolicyViolations == nil {
		t.Error("expected trace to have policy violations in observe mode")
	}
	if traces[0].StatusCode != 200 {
		t.Errorf("expected trace status 200, got %d", traces[0].StatusCode)
	}
}

// --- Test 3: PII redaction (enforce + redact action) ---

func TestPolicyPIIRedaction(t *testing.T) {
	var receivedBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		resp := `{"id":"chatcmpl-test","model":"gpt-4o","usage":{"prompt_tokens":10,"completion_tokens":5},"choices":[{"message":{"role":"assistant","content":"ok"}}]}`
		w.Write([]byte(resp))
	}))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	yamlPolicy := `
rules:
  - name: redact-pii
    type: pii_redaction
    mode: enforce
    config:
      action: redact
`
	engine := newTestPolicyEngine(t, yamlPolicy, store)
	srv.SetPolicyEngine(engine)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"email me at test@example.com please"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// PII redaction with action=redact should NOT block — it modifies and forwards
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify: upstream received redacted body
	if strings.Contains(receivedBody, "test@example.com") {
		t.Error("upstream should have received redacted body, but email was not redacted")
	}
	if !strings.Contains(receivedBody, "[REDACTED]") {
		t.Error("upstream body should contain [REDACTED] placeholder")
	}
}

// --- Test 4: No policy engine (nil) — v1 behavior unchanged ---

func TestNoPolicyEngine(t *testing.T) {
	upstream := httptest.NewServer(mockOpenAIUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)
	// Do NOT set a policy engine — should be nil

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "chatcmpl-test123") {
		t.Fatalf("expected normal response, got: %s", body)
	}

	time.Sleep(200 * time.Millisecond)
	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].PolicyViolations != nil {
		t.Error("expected no policy violations when engine is nil")
	}
}

// --- Test 5: Anthropic error format ---

func TestPolicyBlocksAnthropicFormat(t *testing.T) {
	upstream := httptest.NewServer(mockAnthropicUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("anthropic", upstream.URL)

	yamlPolicy := `
rules:
  - name: tiny-limit
    type: token_limit
    mode: enforce
    config:
      max_tokens: 1
`
	engine := newTestPolicyEngine(t, yamlPolicy, store)
	srv.SetPolicyEngine(engine)

	reqBody := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hello world this is a test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("x-api-key", "sk-ant-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Verify: status 400
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify: X-Corvade-Policy-Violation header present
	if rec.Header().Get("X-Corvade-Policy-Violation") != "true" {
		t.Error("expected X-Corvade-Policy-Violation header to be 'true'")
	}

	// Verify: Anthropic error format
	var errResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Type != "error" {
		t.Errorf("expected top-level type 'error', got %q", errResp.Type)
	}
	if errResp.Error.Type != "policy_violation" {
		t.Errorf("expected error.type 'policy_violation', got %q", errResp.Error.Type)
	}
	if !strings.Contains(errResp.Error.Message, "Corvade policy") {
		t.Errorf("expected message to contain 'Corvade policy', got %q", errResp.Error.Message)
	}
}
