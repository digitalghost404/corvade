package api

import (
	"fmt"
	"net/http"

	"github.com/corvade/corvade/internal/capture"
)

// APIServer wraps the HTTP mux, capture store, and WebSocket hub.
type APIServer struct {
	store *capture.Store
	hub   *Hub
	mux   *http.ServeMux
	port  int
}

// NewAPIServer creates a new APIServer with all routes registered.
func NewAPIServer(store *capture.Store, hub *Hub, port int) *APIServer {
	mux := http.NewServeMux()
	RegisterRoutes(mux, store, hub)
	return &APIServer{store: store, hub: hub, mux: mux, port: port}
}

// Start begins listening on the configured port.
func (s *APIServer) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.mux)
}

// Handler returns the underlying http.Handler for testing.
func (s *APIServer) Handler() http.Handler { return s.mux }
