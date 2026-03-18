package capture

import (
	"fmt"
	"time"
)

func RunRetention(store *Store, retentionDays int) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format(time.RFC3339Nano)

	// Delete graph edges for old traces
	_, err := store.DB().Exec(`
		DELETE FROM graph_edges WHERE session_id IN (
			SELECT DISTINCT session_id FROM traces WHERE created_at < ?
		)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old graph edges: %w", err)
	}

	// Delete graph nodes for old traces
	_, err = store.DB().Exec(`DELETE FROM graph_nodes WHERE trace_id IN (
		SELECT id FROM traces WHERE created_at < ?
	)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old graph nodes: %w", err)
	}

	// Delete old traces
	result, err := store.DB().Exec("DELETE FROM traces WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old traces: %w", err)
	}

	deleted, _ := result.RowsAffected()

	// Clean up empty sessions
	_, err = store.DB().Exec(`DELETE FROM sessions WHERE id NOT IN (
		SELECT DISTINCT session_id FROM traces WHERE session_id IS NOT NULL
	)`)
	if err != nil {
		return deleted, fmt.Errorf("cleaning up empty sessions: %w", err)
	}

	return deleted, nil
}
