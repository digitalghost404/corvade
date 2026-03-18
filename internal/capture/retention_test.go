package capture

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRetentionDeletesOldTraces(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Insert old trace (31 days ago)
	// Note: We insert it directly into the database with the correct timestamp
	// since InsertTrace uses time.Now() internally
	oldTime := time.Now().UTC().AddDate(0, 0, -31).Format(time.RFC3339Nano)
	_, err = store.DB().Exec(`INSERT INTO traces
		(id, session_id, agent, step, provider, model, request, response,
		 status_code, tokens_prompt, tokens_completion, tokens_cached,
		 cost, latency_ms, ttft_ms, api_key_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"01HOLD0000000000000000001", nil, nil, nil, "openai", "gpt-4o",
		`{}`, nil, 200, nil, nil, nil, nil, nil, nil, nil, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert recent trace using normal method
	recent := Trace{
		Provider:   "openai",
		Model:      "gpt-4o",
		Request:    `{}`,
		StatusCode: 200,
	}
	_, err = store.InsertTrace(recent)
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := RunRetention(store, 30)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Recent trace should still exist
	limit := 10
	traces, _ := store.ListTraces(TraceFilter{Limit: &limit})
	if len(traces) != 1 {
		t.Errorf("expected 1 remaining trace, got %d", len(traces))
	}
}

func TestRetentionCleansUpGraphDataAndSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldTime := time.Now().UTC().AddDate(0, 0, -31).Format(time.RFC3339Nano)
	oldTraceID := "01HOLD0000000000000000002"
	oldSessionID := "01SESS0000000000000000001"

	// Insert old session
	_, err = store.DB().Exec(`INSERT INTO sessions
		(id, agent, start_time, trace_count, total_tokens, total_cost, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		oldSessionID, nil, oldTime, 1, 100, 0.05, "completed", oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert old trace linked to the session
	_, err = store.DB().Exec(`INSERT INTO traces
		(id, session_id, provider, model, request, status_code, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		oldTraceID, oldSessionID, "openai", "gpt-4", `{}`, 200, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert graph node for old trace
	_, err = store.DB().Exec(`INSERT INTO graph_nodes
		(id, trace_id, session_id, type, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"01NODE0000000000000000001", oldTraceID, oldSessionID, "llm_call", 0.9, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert graph edge for old session
	_, err = store.DB().Exec(`INSERT INTO graph_edges
		(id, session_id, from_node, to_node, type, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"01EDGE0000000000000000001", oldSessionID, "01NODE0000000000000000001", "01NODE0000000000000000001", "sequence", 0.8, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := RunRetention(store, 30)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted trace, got %d", deleted)
	}

	// Verify graph nodes cleaned up
	var nodeCount int
	store.DB().QueryRow("SELECT COUNT(*) FROM graph_nodes").Scan(&nodeCount)
	if nodeCount != 0 {
		t.Errorf("expected 0 graph nodes after retention, got %d", nodeCount)
	}

	// Verify graph edges cleaned up
	var edgeCount int
	store.DB().QueryRow("SELECT COUNT(*) FROM graph_edges").Scan(&edgeCount)
	if edgeCount != 0 {
		t.Errorf("expected 0 graph edges after retention, got %d", edgeCount)
	}

	// Verify empty session cleaned up
	var sessCount int
	store.DB().QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessCount)
	if sessCount != 0 {
		t.Errorf("expected 0 sessions after retention, got %d", sessCount)
	}
}

func TestRetentionErrorOnClosedDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	_, err = RunRetention(store, 30)
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestRetentionKeepsRecentSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Insert a recent session with a trace
	sessID := "recent-sess"
	sess := Session{StartTime: time.Now().UTC(), Status: "active"}
	// Insert directly to control the session ID
	_, err = store.DB().Exec(`INSERT INTO sessions
		(id, start_time, trace_count, total_tokens, total_cost, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessID, time.Now().UTC().Format(time.RFC3339Nano), 0, 0, 0, "active",
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr := Trace{
		Provider:   "openai",
		Model:      "gpt-4",
		Request:    `{}`,
		StatusCode: 200,
		SessionID:  &sessID,
	}
	_, err = store.InsertTrace(tr)
	if err != nil {
		t.Fatal(err)
	}
	_ = sess // just used for reference

	deleted, err := RunRetention(store, 30)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	var sessCount int
	store.DB().QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessCount)
	if sessCount != 1 {
		t.Errorf("expected 1 session to remain, got %d", sessCount)
	}
}
