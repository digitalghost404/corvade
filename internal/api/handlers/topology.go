package handlers

import (
	"net/http"

	"github.com/corvade/corvade/internal/capture"
)

// TopologyHandler handles the session graph endpoint.
type TopologyHandler struct {
	store *capture.Store
}

// NewTopologyHandler creates a new TopologyHandler.
func NewTopologyHandler(store *capture.Store) *TopologyHandler {
	return &TopologyHandler{store: store}
}

// GraphResponse is the JSON structure for the topology graph endpoint.
type GraphResponse struct {
	Nodes []capture.GraphNode `json:"nodes"`
	Edges []capture.GraphEdge `json:"edges"`
}

// Get handles GET /api/sessions/:id/graph.
func (h *TopologyHandler) Get(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodes, edges, err := h.store.GetSessionGraph(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if nodes == nil {
		nodes = []capture.GraphNode{}
	}
	if edges == nil {
		edges = []capture.GraphEdge{}
	}

	writeJSON(w, http.StatusOK, GraphResponse{Nodes: nodes, Edges: edges})
}
