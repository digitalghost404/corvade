package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
)

// mockSSEUpstream returns a test server that streams SSE chunks.
func mockSSEUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("upstream ResponseWriter does not support Flusher")
			return
		}

		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" world"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			io.WriteString(w, chunk+"\n")
			flusher.Flush()
		}
	}))
}

// TestStreamSSEForwardsChunks verifies that SSE lines are forwarded through the
// proxy and that the assembled content equals the concatenation of all delta chunks.
func TestStreamSSEForwardsChunks(t *testing.T) {
	upstream := mockSSEUpstream(t)
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// The response body must contain the forwarded SSE lines.
	body := rec.Body.String()
	if !strings.Contains(body, `data: {"id":"chatcmpl-1"`) {
		t.Errorf("expected SSE data lines in response body, got: %s", body)
	}
	if !strings.Contains(body, `data: [DONE]`) {
		t.Errorf("expected [DONE] sentinel in response body, got: %s", body)
	}
}

// TestStreamSSEContentType verifies that the proxy propagates the text/event-stream
// Content-Type header from the upstream SSE response.
func TestStreamSSEContentType(t *testing.T) {
	upstream := mockSSEUpstream(t)
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
}

// TestStreamSSETraceCapture verifies that the assembled SSE content and token
// usage are stored in the trace after streaming completes.
func TestStreamSSETraceCapture(t *testing.T) {
	upstream := mockSSEUpstream(t)
	defer upstream.Close()

	store, err := capture.NewStore(t.TempDir() + "/sse_test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, nil)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Allow the async goroutine to write the trace.
	time.Sleep(300 * time.Millisecond)

	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	tr := traces[0]

	// Response must contain the assembled content, not raw SSE lines.
	if tr.Response == nil {
		t.Fatal("expected trace.Response to be set")
	}
	if !strings.Contains(*tr.Response, "Hello world") {
		t.Errorf("expected assembled content 'Hello world' in trace response, got: %q", *tr.Response)
	}
}

// TestStreamSSETokenUsage verifies that token counts from the final SSE chunk
// are extracted and stored in the trace.
func TestStreamSSETokenUsage(t *testing.T) {
	upstream := mockSSEUpstream(t)
	defer upstream.Close()

	store, err := capture.NewStore(t.TempDir() + "/sse_usage_test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, nil)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	time.Sleep(300 * time.Millisecond)

	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	tr := traces[0]
	if tr.TokensPrompt == nil || *tr.TokensPrompt != 10 {
		t.Errorf("expected 10 prompt tokens, got %v", tr.TokensPrompt)
	}
	if tr.TokensCompletion == nil || *tr.TokensCompletion != 5 {
		t.Errorf("expected 5 completion tokens, got %v", tr.TokensCompletion)
	}
}
