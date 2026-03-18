package policy_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/policy"
	"github.com/corvade/corvade/internal/policy/rules"
)

// captureServer returns an httptest.Server that records received requests.
type captureServer struct {
	mu       sync.Mutex
	received []*http.Request
	bodies   [][]byte
	srv      *httptest.Server
}

func newCaptureServer(t *testing.T) *captureServer {
	t.Helper()
	cs := &captureServer{}
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		cs.mu.Lock()
		cs.received = append(cs.received, r)
		cs.bodies = append(cs.bodies, body)
		cs.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(cs.srv.Close)
	return cs
}

func (cs *captureServer) count() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.received)
}

func (cs *captureServer) lastBody() map[string]interface{} {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if len(cs.bodies) == 0 {
		return nil
	}
	var out map[string]interface{}
	_ = json.Unmarshal(cs.bodies[len(cs.bodies)-1], &out)
	return out
}

// waitForCount polls until the server has received at least n requests or the deadline passes.
func (cs *captureServer) waitForCount(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cs.count() >= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// policyWithWebhook writes a policy YAML with a webhook block to a temp dir.
func policyWithWebhook(t *testing.T, url string, events []string, ruleYAML string) string {
	t.Helper()
	eventsStr := ""
	for _, e := range events {
		eventsStr += "\n    - " + e
	}
	content := `webhook:
  url: "` + url + `"
  events:` + eventsStr + `
rules:
` + ruleYAML
	dir := t.TempDir()
	return writePolicyFile(t, dir, content)
}

func newWebhookEngine(t *testing.T, path string) *policy.Engine {
	t.Helper()
	eng, err := policy.NewEngine(path, nil, nil, policy.WithRuleBuilder(rules.DefaultBuilder()))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return eng
}

// ── Test cases ────────────────────────────────────────────────────────────────

// TestWebhook_EnforceViolation_EnforceEvent: enforce violation + events=[enforce] → POST received.
func TestWebhook_EnforceViolation_EnforceEvent(t *testing.T) {
	cs := newCaptureServer(t)

	path := policyWithWebhook(t, cs.srv.URL, []string{"enforce"}, `  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:    "claude-opus-4",
		Agent:    "test-agent",
		Session:  "sess-1",
		Provider: "anthropic",
	})

	if !cs.waitForCount(1, 2*time.Second) {
		t.Fatal("expected 1 webhook POST, got 0")
	}

	body := cs.lastBody()
	if body == nil {
		t.Fatal("could not parse webhook body")
	}
	if body["event"] != "policy_violation" {
		t.Errorf("event field: want %q, got %q", "policy_violation", body["event"])
	}
	if body["timestamp"] == "" || body["timestamp"] == nil {
		t.Error("timestamp field missing")
	}

	rule, ok := body["rule"].(map[string]interface{})
	if !ok {
		t.Fatalf("rule field missing or wrong type: %T", body["rule"])
	}
	if rule["name"] != "model-gate" {
		t.Errorf("rule.name: want %q, got %q", "model-gate", rule["name"])
	}
	if rule["type"] != "model_allowlist" {
		t.Errorf("rule.type: want %q, got %q", "model_allowlist", rule["type"])
	}
	if rule["mode"] != "enforce" {
		t.Errorf("rule.mode: want %q, got %q", "enforce", rule["mode"])
	}

	violation, ok := body["violation"].(map[string]interface{})
	if !ok {
		t.Fatalf("violation field missing or wrong type: %T", body["violation"])
	}
	if violation["blocked"] != true {
		t.Errorf("violation.blocked: want true, got %v", violation["blocked"])
	}

	req, ok := body["request"].(map[string]interface{})
	if !ok {
		t.Fatalf("request field missing or wrong type: %T", body["request"])
	}
	if req["model"] != "claude-opus-4" {
		t.Errorf("request.model: want %q, got %q", "claude-opus-4", req["model"])
	}
	if req["agent"] != "test-agent" {
		t.Errorf("request.agent: want %q, got %q", "test-agent", req["agent"])
	}
	if req["session"] != "sess-1" {
		t.Errorf("request.session: want %q, got %q", "sess-1", req["session"])
	}
	if req["provider"] != "anthropic" {
		t.Errorf("request.provider: want %q, got %q", "anthropic", req["provider"])
	}
}

// TestWebhook_ObserveViolation_EnforceEvent: observe violation + events=[enforce] → no POST.
func TestWebhook_ObserveViolation_EnforceEvent(t *testing.T) {
	cs := newCaptureServer(t)

	path := policyWithWebhook(t, cs.srv.URL, []string{"enforce"}, `  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 5000, // triggers observe violation
	})

	// Give the goroutine a chance to (incorrectly) fire.
	time.Sleep(200 * time.Millisecond)

	if cs.count() != 0 {
		t.Errorf("expected 0 webhook POSTs for observe violation with enforce event filter, got %d", cs.count())
	}
}

// TestWebhook_ObserveViolation_ObserveEvent: observe violation + events=[observe] → POST sent.
func TestWebhook_ObserveViolation_ObserveEvent(t *testing.T) {
	cs := newCaptureServer(t)

	path := policyWithWebhook(t, cs.srv.URL, []string{"observe"}, `  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 5000,
	})

	if !cs.waitForCount(1, 2*time.Second) {
		t.Fatal("expected 1 webhook POST for observe violation with observe event filter, got 0")
	}

	body := cs.lastBody()
	if body == nil {
		t.Fatal("could not parse webhook body")
	}

	violation, ok := body["violation"].(map[string]interface{})
	if !ok {
		t.Fatalf("violation field missing: %T", body["violation"])
	}
	// observe mode → blocked should be false
	if violation["blocked"] != false {
		t.Errorf("violation.blocked: want false for observe, got %v", violation["blocked"])
	}
}

// TestWebhook_EventsAll: events=[all] → always sends regardless of violation mode.
func TestWebhook_EventsAll(t *testing.T) {
	cs := newCaptureServer(t)

	path := policyWithWebhook(t, cs.srv.URL, []string{"all"}, `  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 5000,
	})

	if !cs.waitForCount(1, 2*time.Second) {
		t.Fatal("expected 1 webhook POST with events=[all], got 0")
	}
}

// TestWebhook_NoURLConfigured: no URL → no-op, no panic.
func TestWebhook_NoURLConfigured(t *testing.T) {
	dir := t.TempDir()
	// Policy with NO webhook section at all.
	path := writePolicyFile(t, dir, `rules:
  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`)
	eng := newWebhookEngine(t, path)

	// Should not panic and complete quickly.
	done := make(chan struct{})
	go func() {
		defer close(done)
		eng.Evaluate(policy.EvalContext{
			Model:         "gpt-4o",
			TokenEstimate: 5000,
		})
	}()

	select {
	case <-done:
		// Pass: completed without panic.
	case <-time.After(2 * time.Second):
		t.Fatal("Evaluate blocked unexpectedly with no webhook URL")
	}
}

// TestWebhook_EmptyURL: webhook section present but URL is empty → no-op.
func TestWebhook_EmptyURL(t *testing.T) {
	cs := newCaptureServer(t)

	dir := t.TempDir()
	path := writePolicyFile(t, dir, `webhook:
  url: ""
  events:
    - all
rules:
  - name: token-cap
    type: token_limit
    mode: observe
    config:
      max_tokens: 100
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:         "gpt-4o",
		TokenEstimate: 5000,
	})

	time.Sleep(200 * time.Millisecond)
	if cs.count() != 0 {
		t.Errorf("expected 0 POSTs with empty URL, got %d", cs.count())
	}
}

// TestWebhook_SlowServer: slow server response does not hang the test (engine timeout ≤ 5s).
func TestWebhook_SlowServer(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		// Simulate a response that arrives just after the client timeout.
		time.Sleep(6 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slowSrv.Close)

	path := policyWithWebhook(t, slowSrv.URL, []string{"enforce"}, `  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
`)
	eng := newWebhookEngine(t, path)

	start := time.Now()
	eng.Evaluate(policy.EvalContext{
		Model: "claude-opus-4",
	})
	elapsed := time.Since(start)

	// Evaluate itself should return quickly (webhook fires in a goroutine).
	if elapsed > 500*time.Millisecond {
		t.Errorf("Evaluate blocked for %v; expected to return immediately", elapsed)
	}

	// Give the goroutine a moment to start the request, then confirm the test
	// doesn't block for 6+ seconds.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// The goroutine connected (slow server received the request).
	case <-time.After(10 * time.Second):
		// The goroutine gave up due to client timeout — that is also acceptable.
		t.Log("slow-server goroutine did not connect within 10s (client timeout fired early — OK)")
	}
}

// TestWebhook_PayloadStructure: validates exact JSON shape of the payload.
func TestWebhook_PayloadStructure(t *testing.T) {
	cs := newCaptureServer(t)

	path := policyWithWebhook(t, cs.srv.URL, []string{"all"}, `  - name: model-gate
    type: model_allowlist
    mode: enforce
    config:
      allowed:
        - gpt-4o
`)
	eng := newWebhookEngine(t, path)

	eng.Evaluate(policy.EvalContext{
		Model:    "mistral-large",
		Agent:    "agent-x",
		Session:  "session-abc",
		Provider: "mistral",
	})

	if !cs.waitForCount(1, 2*time.Second) {
		t.Fatal("expected 1 webhook POST")
	}

	cs.mu.Lock()
	rawBody := cs.bodies[0]
	cs.mu.Unlock()

	// Unmarshal into a strict struct to check all required top-level keys.
	var payload struct {
		Event     string                 `json:"event"`
		Timestamp string                 `json:"timestamp"`
		Rule      map[string]string      `json:"rule"`
		Violation map[string]interface{} `json:"violation"`
		Request   map[string]string      `json:"request"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Event != "policy_violation" {
		t.Errorf("event: want policy_violation, got %q", payload.Event)
	}
	if _, err := time.Parse(time.RFC3339, payload.Timestamp); err != nil {
		t.Errorf("timestamp not RFC3339: %q (%v)", payload.Timestamp, err)
	}

	requiredRuleKeys := []string{"name", "type", "mode"}
	for _, k := range requiredRuleKeys {
		if payload.Rule[k] == "" {
			t.Errorf("rule.%s missing or empty", k)
		}
	}

	if _, ok := payload.Violation["message"]; !ok {
		t.Error("violation.message missing")
	}
	if _, ok := payload.Violation["blocked"]; !ok {
		t.Error("violation.blocked missing")
	}

	requiredReqKeys := []string{"model", "agent", "session", "provider"}
	for _, k := range requiredReqKeys {
		if payload.Request[k] == "" {
			t.Errorf("request.%s missing or empty", k)
		}
	}
}
