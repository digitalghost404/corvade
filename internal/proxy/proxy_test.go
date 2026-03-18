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
	"github.com/corvade/corvade/internal/proxy/providers"
)

// --- detectProvider ---

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		path         string
		wantProvider string
		wantPath     string
	}{
		{"/v1/chat/completions", "openai", "/v1/chat/completions"},
		{"/v1", "openai", "/v1"},
		{"/v1/models", "openai", "/v1/models"},
		{"/anthropic/v1/messages", "anthropic", "/v1/messages"},
		{"/anthropic/v1/complete", "anthropic", "/v1/complete"},
		{"/unknown/path", "", "/unknown/path"},
		{"/", "", "/"},
		{"/random", "", "/random"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			provider, upPath := detectProvider(tt.path)
			if provider != tt.wantProvider {
				t.Errorf("detectProvider(%q) provider = %q, want %q", tt.path, provider, tt.wantProvider)
			}
			if upPath != tt.wantPath {
				t.Errorf("detectProvider(%q) path = %q, want %q", tt.path, upPath, tt.wantPath)
			}
		})
	}
}

// --- extractAPIKeyHash ---

func TestExtractAPIKeyHash(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantNil  bool
	}{
		{"bearer auth", map[string]string{"Authorization": "Bearer sk-test"}, false},
		{"x-api-key", map[string]string{"x-api-key": "sk-ant-test"}, false},
		{"no auth", map[string]string{}, true},
		{"empty bearer", map[string]string{"Authorization": "Bearer "}, true},
		{"non-bearer auth", map[string]string{"Authorization": "Basic abc"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			hash := extractAPIKeyHash(req)
			if tt.wantNil && hash != nil {
				t.Errorf("expected nil hash, got %q", *hash)
			}
			if !tt.wantNil && hash == nil {
				t.Error("expected non-nil hash")
			}
			if !tt.wantNil && hash != nil && len(*hash) != 64 {
				t.Errorf("expected 64-char SHA256 hash, got len %d", len(*hash))
			}
		})
	}
}

// --- Unknown route returns 502 ---

func TestProxyUnknownRouteReturns502(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/random/endpoint", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
}

// --- Upstream returns error ---

func TestProxyUpstreamReturnsError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- Upstream closes mid-SSE-stream ---

func TestSSEUpstreamClosesMidStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		io.WriteString(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`+"\n")
		flusher.Flush()
		// Close the connection abruptly (server handler returns)
	}))
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	reqBody := `{"model":"gpt-4o","messages":[],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Errorf("expected partial content 'Hello' in body, got %q", body)
	}
}

// --- streamSSE with no flusher ---

func TestStreamSSENoFlusher(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\ndata: [DONE]\n"
	reader := strings.NewReader(sseData)

	// Use a plain ResponseWriter that does NOT implement Flusher
	w := &nonFlushWriter{header: make(http.Header)}

	assembled, usage, err := streamSSE(reader, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assembled != "Hi" {
		t.Errorf("expected assembled 'Hi', got %q", assembled)
	}
	if usage != nil {
		t.Error("expected nil usage for no usage chunk")
	}
}

// nonFlushWriter implements http.ResponseWriter but NOT http.Flusher.
type nonFlushWriter struct {
	code   int
	header http.Header
	body   strings.Builder
}

func (w *nonFlushWriter) Header() http.Header         { return w.header }
func (w *nonFlushWriter) WriteHeader(code int)         { w.code = code }
func (w *nonFlushWriter) Write(b []byte) (int, error)  { return w.body.Write(b) }

var _ http.ResponseWriter = (*nonFlushWriter)(nil)

// --- streamSSE with cached tokens ---

func TestStreamSSEWithCachedTokens(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":"A"}}]}
data: {"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":50,"prompt_tokens_details":{"cached_tokens":25}}}
data: [DONE]
`
	reader := strings.NewReader(sseData)
	w := httptest.NewRecorder()

	assembled, usage, err := streamSSE(reader, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assembled != "A" {
		t.Errorf("expected 'A', got %q", assembled)
	}
	if usage == nil {
		t.Fatal("expected usage")
	}
	if usage.PromptTokens != 100 {
		t.Errorf("expected 100 prompt tokens, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("expected 50 completion tokens, got %d", usage.CompletionTokens)
	}
	if usage.CachedTokens != 25 {
		t.Errorf("expected 25 cached tokens, got %d", usage.CachedTokens)
	}
}

// --- streamSSE with invalid JSON lines ---

func TestStreamSSEInvalidJSON(t *testing.T) {
	sseData := "data: {not valid json}\ndata: [DONE]\n"
	reader := strings.NewReader(sseData)
	w := httptest.NewRecorder()

	assembled, _, err := streamSSE(reader, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assembled != "" {
		t.Errorf("expected empty assembled for invalid JSON, got %q", assembled)
	}
}

// --- streamSSE with non-data lines ---

func TestStreamSSENonDataLines(t *testing.T) {
	sseData := ": comment\nretry: 1000\nevent: ping\ndata: {\"choices\":[{\"delta\":{\"content\":\"X\"}}]}\ndata: [DONE]\n"
	reader := strings.NewReader(sseData)
	w := httptest.NewRecorder()

	assembled, _, err := streamSSE(reader, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assembled != "X" {
		t.Errorf("expected 'X', got %q", assembled)
	}
}

// --- captureTrace with nil emitter ---

func TestCaptureTraceNilEmitter(t *testing.T) {
	store, err := capture.NewStore(t.TempDir() + "/cap_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, nil) // nil emitter

	srv.captureTrace("openai", "gpt-4", `{"prompt":"hi"}`, `{"text":"hello"}`,
		200, &providers.ResponseInfo{PromptTokens: 10, CompletionTokens: 5}, 100,
		nil, "agent-1", "sess-1", "step-1", nil)

	// Let async write complete (captureTrace is called synchronously in tests)
	time.Sleep(50 * time.Millisecond)

	limit := 10
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	tr := traces[0]
	if tr.Agent == nil || *tr.Agent != "agent-1" {
		t.Errorf("expected agent 'agent-1', got %v", tr.Agent)
	}
	if tr.SessionID == nil || *tr.SessionID != "sess-1" {
		t.Errorf("expected session 'sess-1', got %v", tr.SessionID)
	}
	if tr.Step == nil || *tr.Step != "step-1" {
		t.Errorf("expected step 'step-1', got %v", tr.Step)
	}
}

// --- captureTrace with nil store ---

func TestCaptureTraceNilStore(t *testing.T) {
	srv := &Server{store: nil, calc: nil, emitter: nil}
	// Should not panic
	srv.captureTrace("openai", "gpt-4", `{}`, `{}`, 200, nil, 100, nil, "", "", "", nil)
}

// --- captureTrace with emitter ---

type mockEmitter struct {
	events []string
}

func (m *mockEmitter) Emit(event string, data interface{}) {
	m.events = append(m.events, event)
}

func TestCaptureTraceWithEmitter(t *testing.T) {
	store, err := capture.NewStore(t.TempDir() + "/emit_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	em := &mockEmitter{}
	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, em)

	srv.captureTrace("openai", "gpt-4", `{}`, `{}`, 200,
		&providers.ResponseInfo{PromptTokens: 10, CompletionTokens: 5}, 100,
		nil, "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	if len(em.events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(em.events))
	}
	if em.events[0] != "trace:new" {
		t.Errorf("expected 'trace:new', got %q", em.events[0])
	}
}

// --- captureTrace with cached tokens ---

func TestCaptureTraceWithCachedTokens(t *testing.T) {
	store, err := capture.NewStore(t.TempDir() + "/cache_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	calc := cost.NewCalculator(nil)
	srv := NewServer(store, calc, nil)

	srv.captureTrace("openai", "gpt-4", `{}`, `{}`, 200,
		&providers.ResponseInfo{PromptTokens: 100, CompletionTokens: 50, CachedTokens: 25},
		100, nil, "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	limit := 10
	traces, _ := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].TokensCached == nil || *traces[0].TokensCached != 25 {
		t.Errorf("expected 25 cached tokens, got %v", traces[0].TokensCached)
	}
}

// --- captureTrace with nil respInfo ---

func TestCaptureTraceNilRespInfo(t *testing.T) {
	store, err := capture.NewStore(t.TempDir() + "/nilresp_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, cost.NewCalculator(nil), nil)
	srv.captureTrace("openai", "gpt-4", `{}`, `{}`, 200, nil, 100, nil, "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	limit := 10
	traces, _ := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
}

// --- Proxy with query params ---

func TestProxyForwardsQueryParams(t *testing.T) {
	var gotQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"test","usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream_options=true", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if gotQuery != "stream_options=true" {
		t.Errorf("expected query 'stream_options=true', got %q", gotQuery)
	}
}

// --- No upstream configured for provider ---

func TestProxyNoUpstreamConfigured(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Delete the openai upstream
	delete(srv.upstreamURLs, "openai")

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
}

// --- Malformed JSON body (reqInfo nil fallback) ---

func TestProxyMalformedRequestBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"test","usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", upstream.URL)

	// Send invalid JSON as body — reqInfo will be nil, triggering the fallback
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{not json}`))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (upstream still responds), got %d", rec.Code)
	}
}

// --- Upstream unreachable ---

func TestProxyUpstreamUnreachable(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.SetUpstreamURL("openai", "http://127.0.0.1:1") // unreachable port

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
}

// --- SSE stream error (scanner error from upstream) ---

func TestStreamSSEScannerError(t *testing.T) {
	// Use an errReader that returns an error after some data
	r := &errReader{data: "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n", n: 0}
	w := httptest.NewRecorder()

	assembled, _, err := streamSSE(r, w)
	if err == nil {
		t.Error("expected scanner error")
	}
	if assembled != "Hi" {
		t.Errorf("expected 'Hi', got %q", assembled)
	}
}

type errReader struct {
	data string
	n    int
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.n == 0 {
		r.n++
		n := copy(p, r.data)
		return n, nil
	}
	return 0, io.ErrUnexpectedEOF
}

// --- captureTrace with store insert error (closed DB) ---

func TestCaptureTraceStoreError(t *testing.T) {
	store, err := capture.NewStore(t.TempDir() + "/err_test.db")
	if err != nil {
		t.Fatal(err)
	}
	store.Close() // Close it so InsertTrace fails

	srv := NewServer(store, cost.NewCalculator(nil), nil)
	// Should not panic, just log the error
	srv.captureTrace("openai", "gpt-4", `{}`, `{}`, 200,
		&providers.ResponseInfo{PromptTokens: 10, CompletionTokens: 5}, 100,
		nil, "", "", "", nil)
}

// --- Failed to read request body ---

func TestProxyFailedReadBody(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", &errReaderImmediate{})
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

type errReaderImmediate struct{}

func (r *errReaderImmediate) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// --- Invalid upstream URL (NewRequestWithContext error) ---

func TestProxyInvalidUpstreamURL(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Set an invalid URL that causes http.NewRequestWithContext to fail
	srv.SetUpstreamURL("openai", "://invalid-url\x7f")

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- SSE write error (client disconnect) ---

func TestStreamSSEWriteError(t *testing.T) {
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"A\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\"B\"}}]}\ndata: [DONE]\n"
	reader := strings.NewReader(sseData)

	w := &errWriter{header: make(http.Header)}
	assembled, _, _ := streamSSE(reader, w)

	// Should have stopped after the write error
	if assembled == "AB" {
		t.Error("expected partial assembly due to write error")
	}
}

type errWriter struct {
	header http.Header
	calls  int
}

func (w *errWriter) Header() http.Header        { return w.header }
func (w *errWriter) WriteHeader(code int)        {}
func (w *errWriter) Write(b []byte) (int, error) {
	w.calls++
	if w.calls > 1 {
		return 0, io.ErrClosedPipe
	}
	return len(b), nil
}

var _ http.ResponseWriter = (*errWriter)(nil)

// --- Proxy with Anthropic request parsing ---

func TestProxyAnthropicRequestParsing(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg","usage":{"input_tokens":10,"output_tokens":5}}`))
	}))
	defer upstream.Close()

	srv, store := setupTestServer(t)
	srv.SetUpstreamURL("anthropic", upstream.URL)

	reqBody := `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("x-api-key", "sk-ant-test")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	time.Sleep(200 * time.Millisecond)

	limit := 10
	traces, _ := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].Model != "claude-3-5-sonnet" {
		t.Errorf("expected model 'claude-3-5-sonnet', got %q", traces[0].Model)
	}
}

