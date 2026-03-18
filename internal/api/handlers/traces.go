package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/corvade/corvade/internal/capture"
)

// TracesHandler handles trace-related HTTP endpoints.
type TracesHandler struct {
	store *capture.Store
}

// NewTracesHandler creates a new TracesHandler.
func NewTracesHandler(store *capture.Store) *TracesHandler {
	return &TracesHandler{store: store}
}

// List handles GET /api/traces with optional query params:
// agent, model, limit, offset, search.
func (h *TracesHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	f := capture.TraceFilter{}

	if v := q.Get("agent"); v != "" {
		f.Agent = &v
	}
	if v := q.Get("model"); v != "" {
		f.Model = &v
	}
	if v := q.Get("search"); v != "" {
		f.Search = &v
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			f.Limit = &n
		}
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			f.Offset = &n
		}
	}

	traces, err := h.store.ListTraces(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return empty array instead of null
	if traces == nil {
		traces = []capture.Trace{}
	}

	writeJSON(w, http.StatusOK, traces)
}

// Get handles GET /api/traces/:id.
func (h *TracesHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	trace, err := h.store.GetTrace(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trace not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trace)
}

// writeJSON marshals v as JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
