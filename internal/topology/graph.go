package topology

import (
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/oklog/ulid/v2"
)

// buildNode creates a GraphNode from a trace with the given type and confidence.
func buildNode(trace capture.Trace, nodeType string, confidence float64) capture.GraphNode {
	sessionID := stringOrDefault(trace.SessionID, "unknown")
	tokens := sumTokens(trace.TokensPrompt, trace.TokensCompletion)

	return capture.GraphNode{
		ID:         ulid.Make().String(),
		TraceID:    trace.ID,
		SessionID:  sessionID,
		Type:       nodeType,
		Agent:      trace.Agent,
		Step:       trace.Step,
		Model:      &trace.Model,
		Tokens:     tokens,
		Cost:       trace.Cost,
		LatencyMS:  trace.LatencyMS,
		Confidence: confidence,
		CreatedAt:  time.Now().UTC(),
	}
}

// buildEdge creates a GraphEdge linking two nodes within a session.
func buildEdge(sessionID, fromNode, toNode, edgeType string, confidence float64) capture.GraphEdge {
	return capture.GraphEdge{
		ID:         ulid.Make().String(),
		SessionID:  sessionID,
		FromNode:   fromNode,
		ToNode:     toNode,
		Type:       edgeType,
		Confidence: confidence,
		CreatedAt:  time.Now().UTC(),
	}
}

// stringOrDefault returns s if non-nil, otherwise def.
func stringOrDefault(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

// sumTokens adds two optional int pointers, returning nil if both are nil.
func sumTokens(a, b *int) *int {
	if a == nil && b == nil {
		return nil
	}
	sum := 0
	if a != nil {
		sum += *a
	}
	if b != nil {
		sum += *b
	}
	return &sum
}
