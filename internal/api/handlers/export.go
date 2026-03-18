package handlers

import (
	"net/http"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

// ExportHandler handles the stats/export endpoint.
type ExportHandler struct {
	store *capture.Store
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(store *capture.Store) *ExportHandler {
	return &ExportHandler{store: store}
}

// Stats handles GET /api/stats with optional query params: from, to (RFC3339).
func (h *ExportHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	var from, to *time.Time
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			from = &t
		}
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			to = &t
		}
	}

	stats, err := h.store.GetStats(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
