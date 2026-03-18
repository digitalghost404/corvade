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
