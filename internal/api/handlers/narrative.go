package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/corvade/corvade/internal/capture"
)

// NarrativeSegment represents a single human-readable narrative segment for a graph node.
type NarrativeSegment struct {
	NodeID   string `json:"node_id"`
	Text     string `json:"text"`
	NodeType string `json:"node_type"`
}

// NarrativeResponse is the JSON structure for the narrative endpoint.
type NarrativeResponse struct {
	Segments []NarrativeSegment `json:"segments"`
}

// ptrStringOr returns the dereferenced value of s, or def if s is nil.
func ptrStringOr(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// renderNodeText returns a human-readable description of a graph node.
func renderNodeText(n capture.GraphNode) string {
	agent := ptrStringOr(n.Agent, "agent")
	model := ptrStringOr(n.Model, "unknown model")
	step := ptrStringOr(n.Step, "unknown step")

	latency := 0.0
	if n.LatencyMS != nil {
		latency = float64(*n.LatencyMS) / 1000.0
	}

	cost := 0.0
	if n.Cost != nil {
		cost = *n.Cost
	}

	switch n.Type {
	case "llm_call":
		return fmt.Sprintf("%s asked %s to %s in %.2fs (cost: $%.6f)", agent, model, step, latency, cost)
	case "tool_call":
		return fmt.Sprintf("It called %s in %.2fs", step, latency)
	case "tool_result":
		return fmt.Sprintf("It received the result from %s", step)
	case "decision_point":
		return fmt.Sprintf("It decided to %s", step)
	default:
		return fmt.Sprintf("Node %s (%s)", n.ID, n.Type)
	}
}

// GetNarrative handles GET /api/sessions/:id/narrative.
// It returns a human-readable narrative of the session graph nodes.
// If ?format=markdown is specified, it returns plain text with bullet points.
func (h *TopologyHandler) GetNarrative(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodes, _, err := h.store.GetSessionGraph(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	segments := make([]NarrativeSegment, 0, len(nodes))
	for _, n := range nodes {
		segments = append(segments, NarrativeSegment{
			NodeID:   n.ID,
			Text:     renderNodeText(n),
			NodeType: n.Type,
		})
	}

	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		var sb strings.Builder
		for _, seg := range segments {
			sb.WriteString("- ")
			sb.WriteString(seg.Text)
			sb.WriteString("\n")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sb.String())) //nolint:errcheck
		return
	}

	writeJSON(w, http.StatusOK, NarrativeResponse{Segments: segments})
}

// EnhanceNarrative handles POST /api/sessions/:id/narrative/enhance.
// Currently a stub — LLM-enhanced narrative is not yet implemented.
func (h *TopologyHandler) EnhanceNarrative(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "LLM-enhanced narrative coming in a future release",
	})
}
