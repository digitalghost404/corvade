package topology

import (
	"github.com/corvade/corvade/internal/capture"
)

// Engine infers a directed graph (nodes + edges) from a list of captured traces.
type Engine struct {
	timingGapMS int
}

// NewEngine creates an inference engine with the given timing gap threshold in milliseconds.
func NewEngine(timingGapMS int) *Engine {
	return &Engine{timingGapMS: timingGapMS}
}

// Infer takes a list of traces and produces nodes and edges forming a directed graph.
// Three inference strategies are applied in priority order:
//  1. Tool call ID tracking (confidence ~0.9)
//  2. Conversation fingerprinting (confidence ~0.7)
//  3. Timing gaps (confidence ~0.5)
func (e *Engine) Infer(traces []capture.Trace) ([]capture.GraphNode, []capture.GraphEdge) {
	if len(traces) == 0 {
		return nil, nil
	}

	// Build nodes for each trace.
	nodes := make([]capture.GraphNode, len(traces))
	for i, t := range traces {
		nodes[i] = buildNode(t, inferNodeType(t), 1.0)
	}

	// Track which (from, to) pairs already have edges.
	type edgeKey struct{ from, to int }
	connected := make(map[edgeKey]bool)

	var edges []capture.GraphEdge

	// Build index: tool_call_id -> trace index (for responses that produced them).
	toolCallToTrace := make(map[string]int)
	for i, t := range traces {
		if t.Response == nil {
			continue
		}
		for _, id := range ExtractToolCallIDs(*t.Response) {
			toolCallToTrace[id] = i
		}
	}

	// Strategy 1: Tool call ID tracking (highest confidence ~0.9).
	for i, t := range traces {
		refIDs := ExtractReferencedToolCallIDs(t.Request)
		for _, refID := range refIDs {
			if fromIdx, ok := toolCallToTrace[refID]; ok && fromIdx != i {
				key := edgeKey{fromIdx, i}
				if !connected[key] {
					connected[key] = true
					sessionID := stringOrDefault(t.SessionID, "unknown")
					edges = append(edges, buildEdge(sessionID, nodes[fromIdx].ID, nodes[i].ID, "triggered", 0.9))
				}
			}
		}
	}

	// Strategy 2: Conversation fingerprinting (medium confidence ~0.7).
	for i := 1; i < len(traces); i++ {
		for j := 0; j < i; j++ {
			key := edgeKey{j, i}
			if connected[key] {
				continue
			}
			if traces[j].Response == nil {
				continue
			}
			if ContainsAssistantResponse(traces[i].Request, *traces[j].Response) {
				connected[key] = true
				sessionID := stringOrDefault(traces[i].SessionID, "unknown")
				edges = append(edges, buildEdge(sessionID, nodes[j].ID, nodes[i].ID, "triggered", 0.7))
			}
		}
	}

	// Strategy 3: Timing gaps (lowest confidence ~0.5).
	for i := 1; i < len(traces); i++ {
		// Check if this node already has any incoming edge.
		hasIncoming := false
		for j := 0; j < i; j++ {
			if connected[edgeKey{j, i}] {
				hasIncoming = true
				break
			}
		}
		if hasIncoming {
			continue
		}

		// Find the closest preceding trace within the timing threshold.
		prev := i - 1
		gapMS := traces[i].CreatedAt.Sub(traces[prev].CreatedAt).Milliseconds()
		if gapMS >= 0 && gapMS <= int64(e.timingGapMS) {
			key := edgeKey{prev, i}
			if !connected[key] {
				connected[key] = true
				sessionID := stringOrDefault(traces[i].SessionID, "unknown")
				confidence := 0.6 - (float64(gapMS)/float64(e.timingGapMS))*0.1
				edges = append(edges, buildEdge(sessionID, nodes[prev].ID, nodes[i].ID, "timing", confidence))
			}
		}
	}

	return nodes, edges
}

// inferNodeType determines the node type based on the trace.
func inferNodeType(t capture.Trace) string {
	if t.Response == nil {
		return "request"
	}
	ids := ExtractToolCallIDs(*t.Response)
	if len(ids) > 0 {
		return "tool_call"
	}
	return "completion"
}
