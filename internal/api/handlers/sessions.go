package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/corvade/corvade/internal/capture"
)

// SessionsHandler handles session-related HTTP endpoints.
type SessionsHandler struct {
	store *capture.Store
}

// NewSessionsHandler creates a new SessionsHandler.
func NewSessionsHandler(store *capture.Store) *SessionsHandler {
	return &SessionsHandler{store: store}
}

// List handles GET /api/sessions with optional query params: agent, limit, offset.
func (h *SessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	f := capture.SessionFilter{}

	if v := q.Get("agent"); v != "" {
		f.Agent = &v
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

	sessions, err := h.store.ListSessionsFiltered(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if sessions == nil {
		sessions = []capture.Session{}
	}

	writeJSON(w, http.StatusOK, sessions)
}

// Get handles GET /api/sessions/:id.
func (h *SessionsHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := h.store.GetSession(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, session)
}
