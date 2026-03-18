package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
)

// mockOpenAIUpstream returns a handler that echoes back a valid OpenAI response
func mockOpenAIUpstream(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify corvade headers are NOT forwarded
		if r.Header.Get("X-Corvade-Agent") != "" {
			t.Error("X-Corvade-Agent header was forwarded to upstream")
		}
		if r.Header.Get("X-Corvade-Session") != "" {
			t.Error("X-Corvade-Session header was forwarded to upstream")
		}
		if r.Header.Get("X-Corvade-Step") != "" {
			t.Error("X-Corvade-Step header was forwarded to upstream")
		}

		resp := map[string]interface{}{
			"id":    "chatcmpl-test123",
			"model": "gpt-4o",
			"usage": map[string]int{
				"prompt_tokens":     100,
				"completion_tokens": 50,
			},
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from mock OpenAI!",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

// mockAnthropicUpstream returns a handler that echoes back a valid Anthropic response
func mockAnthropicUpstream(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":    "msg_test123",
			"model": "claude-sonnet-4-6",
			"type":  "message",
			"usage": map[string]int{
				"input_tokens":  200,
				"output_tokens": 80,
			},
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Hello from mock Anthropic!",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

func setupTestServer(t *testing.T) (*Server, *capture.Store) {
	t.Helper()
	store, err := capture.NewStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, nil)
	return srv, store
}

func TestProxyForwardsOpenAIRequest(t *testing.T) {
	// Start mock upstream
	upstream := httptest.NewServer(mockOpenAIUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	// Build an OpenAI-style request
	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Verify proxy forwarded the response
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "chatcmpl-test123") {
		t.Fatalf("response body missing expected content: %s", body)
	}

	// Wait for async capture
	time.Sleep(200 * time.Millisecond)

	// Verify trace was captured
	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	tr := traces[0]
	if tr.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", tr.Provider)
	}
	if tr.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", tr.Model)
	}
	if tr.TokensPrompt == nil || *tr.TokensPrompt != 100 {
		t.Errorf("expected 100 prompt tokens, got %v", tr.TokensPrompt)
	}
	if tr.TokensCompletion == nil || *tr.TokensCompletion != 50 {
		t.Errorf("expected 50 completion tokens, got %v", tr.TokensCompletion)
	}
	if tr.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", tr.StatusCode)
	}
	if tr.APIKeyHash == nil || *tr.APIKeyHash == "" {
		t.Error("expected non-empty API key hash")
	}
}

func TestProxyForwardsAnthropicRequest(t *testing.T) {
	upstream := httptest.NewServer(mockAnthropicUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("anthropic", upstream.URL)

	reqBody := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("x-api-key", "sk-ant-test-key-12345")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "msg_test123") {
		t.Fatalf("response body missing expected content: %s", body)
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

	tr := traces[0]
	if tr.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", tr.Provider)
	}
	if tr.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", tr.Model)
	}
	if tr.TokensPrompt == nil || *tr.TokensPrompt != 200 {
		t.Errorf("expected 200 prompt tokens, got %v", tr.TokensPrompt)
	}
	if tr.TokensCompletion == nil || *tr.TokensCompletion != 80 {
		t.Errorf("expected 80 completion tokens, got %v", tr.TokensCompletion)
	}
	if tr.APIKeyHash == nil || *tr.APIKeyHash == "" {
		t.Error("expected non-empty API key hash")
	}
}

func TestProxyExtractsCorvadeHeaders(t *testing.T) {
	upstream := httptest.NewServer(mockOpenAIUpstream(t))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key-12345")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Corvade-Agent", "my-agent")
	req.Header.Set("X-Corvade-Session", "sess-abc")
	req.Header.Set("X-Corvade-Step", "step-1")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
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

	tr := traces[0]
	// Verify corvade headers were captured in the trace
	if tr.Agent == nil || *tr.Agent != "my-agent" {
		t.Errorf("expected agent 'my-agent', got %v", tr.Agent)
	}
	if tr.SessionID == nil || *tr.SessionID != "sess-abc" {
		t.Errorf("expected session_id 'sess-abc', got %v", tr.SessionID)
	}
	if tr.Step == nil || *tr.Step != "step-1" {
		t.Errorf("expected step 'step-1', got %v", tr.Step)
	}

	// The mock upstream verifies headers were stripped (see mockOpenAIUpstream)
}
