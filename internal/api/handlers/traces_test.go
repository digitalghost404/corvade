package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/corvade/corvade/internal/api/handlers"
	"github.com/corvade/corvade/internal/capture"
)

func newTestStore(t *testing.T) *capture.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := capture.NewStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func insertTestTrace(t *testing.T, store *capture.Store) string {
	t.Helper()
	agent := "test-agent"
	cost := 0.05
	prompt := 100
	completion := 50
	tr := capture.Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{"prompt":"hello"}`,
		StatusCode:       200,
		Agent:            &agent,
		Cost:             &cost,
		TokensPrompt:    &prompt,
		TokensCompletion: &completion,
	}
	id, err := store.InsertTrace(tr)
	if err != nil {
		t.Fatalf("insert trace: %v", err)
	}
	return id
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestListTracesHandler(t *testing.T) {
	store := newTestStore(t)
	_ = insertTestTrace(t, store)

	h := handlers.NewTracesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/traces?limit=10", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var traces []capture.Trace
	if err := json.NewDecoder(w.Body).Decode(&traces); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	if traces[0].Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", traces[0].Provider)
	}
}

func TestGetTraceHandler(t *testing.T) {
	store := newTestStore(t)
	id := insertTestTrace(t, store)

	h := handlers.NewTracesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/traces/"+id, nil)
	w := httptest.NewRecorder()

	h.Get(w, req, id)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var trace capture.Trace
	if err := json.NewDecoder(w.Body).Decode(&trace); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if trace.ID != id {
		t.Errorf("expected trace ID %q, got %q", id, trace.ID)
	}
}

func TestGetTraceNotFound(t *testing.T) {
	store := newTestStore(t)

	h := handlers.NewTracesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/traces/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Get(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestListTracesFilterByAgent(t *testing.T) {
	store := newTestStore(t)
	_ = insertTestTrace(t, store) // agent = "test-agent"

	h := handlers.NewTracesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/traces?agent=test-agent", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var traces []capture.Trace
	if err := json.NewDecoder(w.Body).Decode(&traces); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	// Filter by non-matching agent
	req2 := httptest.NewRequest(http.MethodGet, "/api/traces?agent=other-agent", nil)
	w2 := httptest.NewRecorder()
	h.List(w2, req2)

	var traces2 []capture.Trace
	json.NewDecoder(w2.Body).Decode(&traces2)

	if len(traces2) != 0 {
		t.Fatalf("expected 0 traces for non-matching agent, got %d", len(traces2))
	}
}

func TestListTracesEmpty(t *testing.T) {
	store := newTestStore(t)

	h := handlers.NewTracesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Should return empty array, not null
	body := w.Body.String()
	if body != "[]\n" {
		t.Errorf("expected empty array '[]\\n', got %q", body)
	}
}

func TestStatsHandler(t *testing.T) {
	store := newTestStore(t)
	_ = insertTestTrace(t, store)
	_ = insertTestTrace(t, store)

	h := handlers.NewExportHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var stats capture.Stats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if stats.TraceCount != 2 {
		t.Errorf("expected trace_count 2, got %d", stats.TraceCount)
	}
	if stats.TotalTokens != 300 { // 2 * (100 + 50)
		t.Errorf("expected total_tokens 300, got %d", stats.TotalTokens)
	}
	if stats.TotalCost != 0.1 { // 2 * 0.05
		t.Errorf("expected total_cost 0.1, got %f", stats.TotalCost)
	}
}
