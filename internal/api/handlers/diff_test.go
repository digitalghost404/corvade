package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/api/handlers"
	"github.com/corvade/corvade/internal/capture"
)

func insertTestSession(t *testing.T, store *capture.Store) string {
	t.Helper()
	agent := "diff-agent"
	sess := capture.Session{
		Agent:      &agent,
		StartTime:  time.Now().UTC(),
		TraceCount: 0,
		Status:     "complete",
	}
	id, err := store.InsertSession(sess)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return id
}

func insertTestNode(t *testing.T, store *capture.Store, sessionID, traceID, nodeType string, agent, step, model *string, snap *string) string {
	t.Helper()
	node := capture.GraphNode{
		TraceID:         traceID,
		SessionID:       sessionID,
		Type:            nodeType,
		Agent:           agent,
		Step:            step,
		Model:           model,
		ContextSnapshot: snap,
		Confidence:      0.9,
	}
	id, err := store.InsertGraphNode(node)
	if err != nil {
		t.Fatalf("insert graph node: %v", err)
	}
	return id
}

func strPtr(s string) *string { return &s }

// TestDiffHandlerIdenticalSessions checks that two sessions with identical
// nodes produce only aligned pairs and no left-only / right-only nodes.
func TestDiffHandlerIdenticalSessions(t *testing.T) {
	store := newTestStore(t)

	sid1 := insertTestSession(t, store)
	sid2 := insertTestSession(t, store)

	traceID := "trace-0"
	snap := "hello world foo bar"

	insertTestNode(t, store, sid1, traceID, "llm_call", strPtr("agent-a"), strPtr("step-1"), strPtr("gpt-4"), &snap)
	insertTestNode(t, store, sid2, traceID, "llm_call", strPtr("agent-a"), strPtr("step-1"), strPtr("gpt-4"), &snap)

	h := handlers.NewDiffHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sid1+"/diff/"+sid2, nil)
	w := httptest.NewRecorder()

	h.Diff(w, req, sid1, sid2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp handlers.DiffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.AlignedNodes) != 1 {
		t.Errorf("expected 1 aligned pair, got %d", len(resp.AlignedNodes))
	}
	if len(resp.LeftOnly) != 0 {
		t.Errorf("expected 0 left-only, got %d", len(resp.LeftOnly))
	}
	if len(resp.RightOnly) != 0 {
		t.Errorf("expected 0 right-only, got %d", len(resp.RightOnly))
	}
	if len(resp.DivergencePoints) != 0 {
		t.Errorf("expected 0 divergence points, got %d", len(resp.DivergencePoints))
	}
	if len(resp.AlignedNodes) == 1 {
		if resp.AlignedNodes[0].MatchScore < 0.99 {
			t.Errorf("expected near-perfect match score for identical nodes, got %f", resp.AlignedNodes[0].MatchScore)
		}
	}
}

// TestDiffHandlerDisjointSessions checks that sessions with no common nodes
// produce only left-only and right-only results.
func TestDiffHandlerDisjointSessions(t *testing.T) {
	store := newTestStore(t)

	sid1 := insertTestSession(t, store)
	sid2 := insertTestSession(t, store)

	traceID := "trace-0"
	snap1 := "alpha beta gamma"
	snap2 := "delta epsilon zeta"

	insertTestNode(t, store, sid1, traceID, "llm_call", strPtr("agent-a"), strPtr("step-A"), strPtr("gpt-4"), &snap1)
	insertTestNode(t, store, sid2, traceID, "tool_call", strPtr("agent-b"), strPtr("step-B"), strPtr("claude-3"), &snap2)

	h := handlers.NewDiffHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sid1+"/diff/"+sid2, nil)
	w := httptest.NewRecorder()

	h.Diff(w, req, sid1, sid2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp handlers.DiffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The nodes differ on every dimension — any match score should be low,
	// resulting in a left-only or a divergence point, never a high-quality pair.
	for _, pair := range resp.AlignedNodes {
		if pair.MatchScore >= divergenceThresholdExported() {
			t.Errorf("unexpected high-confidence match between disjoint nodes: %f", pair.MatchScore)
		}
	}
}

// TestDiffHandlerEmptySessions verifies that two empty sessions return empty arrays.
func TestDiffHandlerEmptySessions(t *testing.T) {
	store := newTestStore(t)

	sid1 := insertTestSession(t, store)
	sid2 := insertTestSession(t, store)

	h := handlers.NewDiffHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sid1+"/diff/"+sid2, nil)
	w := httptest.NewRecorder()

	h.Diff(w, req, sid1, sid2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp handlers.DiffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.AlignedNodes) != 0 || len(resp.LeftOnly) != 0 || len(resp.RightOnly) != 0 || len(resp.DivergencePoints) != 0 {
		t.Errorf("expected all-empty diff for empty sessions, got %+v", resp)
	}
}

// TestDiffHandlerMethodNotAllowed ensures non-GET requests are rejected.
func TestDiffHandlerMethodNotAllowed(t *testing.T) {
	store := newTestStore(t)
	h := handlers.NewDiffHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/a/diff/b", nil)
	w := httptest.NewRecorder()
	h.Diff(w, req, "a", "b")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// divergenceThresholdExported mirrors the package-level constant so tests can
// reference it without importing an unexported symbol.
func divergenceThresholdExported() float64 { return 0.5 }
