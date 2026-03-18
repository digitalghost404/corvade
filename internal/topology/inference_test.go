package topology

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

func loadFixture(t *testing.T, path string) []capture.Trace {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	var traces []capture.Trace
	if err := json.Unmarshal(data, &traces); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
	return traces
}

func TestInferToolCallChain(t *testing.T) {
	traces := loadFixture(t, "../../testdata/tool-call-chain.json")
	engine := NewEngine(2000)
	nodes, edges := engine.Infer(traces)

	if len(nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(nodes))
	}
	if len(edges) < 1 {
		t.Fatalf("expected at least 1 edge, got %d", len(edges))
	}

	// The edge should be "triggered" from trace 1 -> trace 2 via tool_call_id tracking.
	found := false
	for _, e := range edges {
		if e.Type == "triggered" && e.FromNode == nodes[0].ID && e.ToNode == nodes[1].ID {
			found = true
			if e.Confidence < 0.8 {
				t.Errorf("expected high confidence for tool call edge, got %f", e.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected a triggered edge from trace 1 to trace 2")
	}
}

func TestInferToolCallChain_NodeTypes(t *testing.T) {
	traces := loadFixture(t, "../../testdata/tool-call-chain.json")
	engine := NewEngine(2000)
	nodes, _ := engine.Infer(traces)

	if nodes[0].Type != "tool_call" {
		t.Errorf("expected first node type 'tool_call', got %q", nodes[0].Type)
	}
	if nodes[1].Type != "completion" {
		t.Errorf("expected second node type 'completion', got %q", nodes[1].Type)
	}
}

func TestInferToolCallChain_NoDuplicateEdges(t *testing.T) {
	traces := loadFixture(t, "../../testdata/tool-call-chain.json")
	engine := NewEngine(5000) // Large window to also trigger timing
	nodes, edges := engine.Infer(traces)

	// There should be exactly 1 edge (tool call tracking wins, others skip).
	edgeCount := 0
	for _, e := range edges {
		if e.FromNode == nodes[0].ID && e.ToNode == nodes[1].ID {
			edgeCount++
		}
	}
	if edgeCount != 1 {
		t.Errorf("expected exactly 1 edge between nodes, got %d", edgeCount)
	}
}

func TestInferEmpty(t *testing.T) {
	engine := NewEngine(2000)
	nodes, edges := engine.Infer(nil)
	if nodes != nil || edges != nil {
		t.Error("expected nil results for empty input")
	}
}

func TestExtractToolCallIDs_OpenAI(t *testing.T) {
	body := `{"choices":[{"message":{"tool_calls":[{"id":"call_abc"},{"id":"call_def"}]}}]}`
	ids := ExtractToolCallIDs(body)
	if len(ids) != 2 || ids[0] != "call_abc" || ids[1] != "call_def" {
		t.Errorf("unexpected tool call IDs: %v", ids)
	}
}

func TestExtractToolCallIDs_Anthropic(t *testing.T) {
	body := `{"content":[{"type":"tool_use","id":"toolu_123"},{"type":"text","text":"hello"}]}`
	ids := ExtractToolCallIDs(body)
	if len(ids) != 1 || ids[0] != "toolu_123" {
		t.Errorf("unexpected tool call IDs: %v", ids)
	}
}

func TestExtractReferencedToolCallIDs(t *testing.T) {
	body := `{"messages":[{"role":"user","content":"hi"},{"role":"tool","tool_call_id":"call_abc","content":"result"}]}`
	ids := ExtractReferencedToolCallIDs(body)
	if len(ids) != 1 || ids[0] != "call_abc" {
		t.Errorf("unexpected referenced IDs: %v", ids)
	}
}

func TestFingerprintMessages(t *testing.T) {
	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`
	fp := FingerprintMessages(body)
	if fp == "" {
		t.Error("expected non-empty fingerprint")
	}
	// Same input should produce same fingerprint.
	fp2 := FingerprintMessages(body)
	if fp != fp2 {
		t.Error("fingerprint should be deterministic")
	}
}

func TestContainsAssistantResponse(t *testing.T) {
	responseA := `{"choices":[{"message":{"content":"Here are the results..."}}]}`
	requestB := `{"messages":[{"role":"assistant","content":"Here are the results..."},{"role":"user","content":"tell me more"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected requestB to contain responseA's assistant content")
	}

	unrelatedReq := `{"messages":[{"role":"user","content":"something completely different"}]}`
	if ContainsAssistantResponse(unrelatedReq, responseA) {
		t.Error("expected unrelated request to NOT contain responseA's content")
	}
}

func TestStringOrDefault(t *testing.T) {
	s := "hello"
	if stringOrDefault(&s, "default") != "hello" {
		t.Error("expected 'hello'")
	}
	if stringOrDefault(nil, "default") != "default" {
		t.Error("expected 'default'")
	}
}

func TestSumTokens(t *testing.T) {
	a, b := 10, 20
	result := sumTokens(&a, &b)
	if result == nil || *result != 30 {
		t.Errorf("expected 30, got %v", result)
	}

	result = sumTokens(nil, nil)
	if result != nil {
		t.Error("expected nil for nil inputs")
	}

	result = sumTokens(&a, nil)
	if result == nil || *result != 10 {
		t.Errorf("expected 10, got %v", result)
	}

	result = sumTokens(nil, &b)
	if result == nil || *result != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

// --- Fingerprint tests ---

func TestFingerprintMessagesEmptyBody(t *testing.T) {
	if fp := FingerprintMessages(""); fp != "" {
		t.Errorf("expected empty fingerprint for empty body, got %q", fp)
	}
}

func TestFingerprintMessagesMalformedJSON(t *testing.T) {
	if fp := FingerprintMessages("{bad json"); fp != "" {
		t.Errorf("expected empty fingerprint for malformed JSON, got %q", fp)
	}
}

func TestFingerprintMessagesNoMessagesField(t *testing.T) {
	if fp := FingerprintMessages(`{"model":"gpt-4"}`); fp != "" {
		t.Errorf("expected empty fingerprint when no messages field, got %q", fp)
	}
}

// --- ContainsAssistantResponse edge cases ---

func TestContainsAssistantResponseEmptyResponse(t *testing.T) {
	if ContainsAssistantResponse(`{"messages":[]}`, "") {
		t.Error("expected false for empty response")
	}
}

func TestContainsAssistantResponseAnthropicFormat(t *testing.T) {
	responseA := `{"content":[{"type":"text","text":"Here is my analysis of the data"}]}`
	requestB := `{"messages":[{"role":"assistant","content":"Here is my analysis of the data"},{"role":"user","content":"more"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected Anthropic text content to match")
	}
}

func TestContainsAssistantResponseAnthropicToolUse(t *testing.T) {
	responseA := `{"content":[{"type":"tool_use","id":"toolu_xyz123"}]}`
	requestB := `{"messages":[{"role":"tool","tool_call_id":"toolu_xyz123","content":"result"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected Anthropic tool_use ID to match")
	}
}

func TestContainsAssistantResponseNoContent(t *testing.T) {
	responseA := `{"choices":[]}`
	requestB := `{"messages":[{"role":"user","content":"hello"}]}`

	if ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected false when response has no content")
	}
}

func TestContainsAssistantResponseOpenAIToolCallID(t *testing.T) {
	responseA := `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_xyz"}]}}]}`
	requestB := `{"messages":[{"role":"tool","tool_call_id":"call_xyz","content":"result"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected OpenAI tool_call ID to match")
	}
}

func TestContainsAssistantResponseLongContent(t *testing.T) {
	// Content longer than 50 chars should be truncated to 50 for matching
	longContent := "This is a very long response that exceeds fifty characters in total length"
	responseA := `{"choices":[{"message":{"content":"` + longContent + `"}}]}`
	// Request only needs to contain the first 50 chars
	requestB := `{"messages":[{"role":"assistant","content":"` + longContent[:50] + `"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected truncated long content to match")
	}
}

func TestContainsAssistantResponseAnthropicLongText(t *testing.T) {
	longText := "This is a very long Anthropic response that exceeds the fifty character limit for markers"
	responseA := `{"content":[{"type":"text","text":"` + longText + `"}]}`
	requestB := `{"messages":[{"role":"assistant","content":"` + longText[:50] + `"}]}`

	if !ContainsAssistantResponse(requestB, responseA) {
		t.Error("expected truncated Anthropic text to match")
	}
}

// --- ToolCall extraction edge cases ---

func TestExtractToolCallIDsEmptyBody(t *testing.T) {
	ids := ExtractToolCallIDs("")
	if len(ids) != 0 {
		t.Errorf("expected no IDs for empty body, got %v", ids)
	}
}

func TestExtractToolCallIDsMalformedJSON(t *testing.T) {
	ids := ExtractToolCallIDs("{bad json")
	if len(ids) != 0 {
		t.Errorf("expected no IDs for malformed JSON, got %v", ids)
	}
}

func TestExtractToolCallIDsAnthropicMixedContent(t *testing.T) {
	body := `{"content":[{"type":"text","text":"thinking..."},{"type":"tool_use","id":"toolu_1"},{"type":"tool_use","id":"toolu_2"}]}`
	ids := ExtractToolCallIDs(body)
	if len(ids) != 2 {
		t.Errorf("expected 2 Anthropic tool IDs, got %d: %v", len(ids), ids)
	}
}

func TestExtractToolCallIDsEmptyArrays(t *testing.T) {
	body := `{"choices":[{"message":{"tool_calls":[]}}]}`
	ids := ExtractToolCallIDs(body)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs for empty tool_calls, got %v", ids)
	}
}

func TestExtractToolCallIDsEmptyID(t *testing.T) {
	body := `{"choices":[{"message":{"tool_calls":[{"id":""}]}}]}`
	ids := ExtractToolCallIDs(body)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs when ID is empty string, got %v", ids)
	}
}

func TestExtractReferencedToolCallIDsMalformed(t *testing.T) {
	ids := ExtractReferencedToolCallIDs("{bad json")
	if ids != nil {
		t.Errorf("expected nil for malformed JSON, got %v", ids)
	}
}

func TestExtractReferencedToolCallIDsNoToolMessages(t *testing.T) {
	body := `{"messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]}`
	ids := ExtractReferencedToolCallIDs(body)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs when no tool messages, got %v", ids)
	}
}

// --- Inference engine tests ---

func TestInferSingleTrace(t *testing.T) {
	engine := NewEngine(2000)
	resp := `{"choices":[{"message":{"content":"hello"}}]}`
	traces := []capture.Trace{
		{
			ID:        "trace-1",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"hi"}]}`,
			Response:  &resp,
			CreatedAt: time.Now().UTC(),
		},
	}
	nodes, edges := engine.Infer(traces)
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges for single trace, got %d", len(edges))
	}
	if nodes[0].Type != "completion" {
		t.Errorf("expected node type 'completion', got %q", nodes[0].Type)
	}
}

func TestInferTimingGap(t *testing.T) {
	engine := NewEngine(2000)
	now := time.Now().UTC()
	resp1 := `{"choices":[{"message":{"content":"first"}}]}`
	resp2 := `{"choices":[{"message":{"content":"second"}}]}`

	traces := []capture.Trace{
		{
			ID:        "trace-1",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"a"}]}`,
			Response:  &resp1,
			CreatedAt: now,
		},
		{
			ID:        "trace-2",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"b"}]}`,
			Response:  &resp2,
			CreatedAt: now.Add(500 * time.Millisecond),
		},
	}

	nodes, edges := engine.Infer(traces)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	// Should get a timing edge since no tool-call or fingerprint match
	found := false
	for _, e := range edges {
		if e.Type == "timing" {
			found = true
			if e.Confidence < 0.5 || e.Confidence > 0.6 {
				t.Errorf("timing edge confidence out of range: %f", e.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected a timing edge between traces within gap threshold")
	}
}

func TestInferTimingGapExceedsThreshold(t *testing.T) {
	engine := NewEngine(100) // very small gap
	now := time.Now().UTC()
	resp := `{"choices":[{"message":{"content":"x"}}]}`

	traces := []capture.Trace{
		{
			ID:        "trace-1",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"a"}]}`,
			Response:  &resp,
			CreatedAt: now,
		},
		{
			ID:        "trace-2",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"b"}]}`,
			Response:  &resp,
			CreatedAt: now.Add(5 * time.Second), // Way beyond 100ms gap
		},
	}

	_, edges := engine.Infer(traces)
	for _, e := range edges {
		if e.Type == "timing" {
			t.Error("should not create timing edge when gap exceeds threshold")
		}
	}
}

func TestInferNodeTypeRequest(t *testing.T) {
	// Trace with nil response should be type "request"
	engine := NewEngine(2000)
	traces := []capture.Trace{
		{
			ID:        "trace-1",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{}`,
			Response:  nil,
			CreatedAt: time.Now().UTC(),
		},
	}
	nodes, _ := engine.Infer(traces)
	if nodes[0].Type != "request" {
		t.Errorf("expected node type 'request' for nil response, got %q", nodes[0].Type)
	}
}

func TestInferConversationFingerprinting(t *testing.T) {
	engine := NewEngine(100) // Small gap to avoid timing edges
	now := time.Now().UTC()

	resp1 := `{"choices":[{"message":{"content":"Here is my analysis of the situation at hand"}}]}`
	traces := []capture.Trace{
		{
			ID:        "trace-1",
			Provider:  "openai",
			Model:     "gpt-4",
			Request:   `{"messages":[{"role":"user","content":"analyze"}]}`,
			Response:  &resp1,
			CreatedAt: now,
		},
		{
			ID:       "trace-2",
			Provider: "openai",
			Model:    "gpt-4",
			// Request contains the assistant response from trace-1
			Request:   `{"messages":[{"role":"user","content":"analyze"},{"role":"assistant","content":"Here is my analysis of the situation at hand"},{"role":"user","content":"more"}]}`,
			Response:  nil,
			CreatedAt: now.Add(10 * time.Second), // Beyond timing gap
		},
	}

	_, edges := engine.Infer(traces)
	found := false
	for _, e := range edges {
		if e.Type == "triggered" && e.Confidence >= 0.7 {
			found = true
		}
	}
	if !found {
		t.Error("expected a fingerprint-based triggered edge")
	}
}

// --- Graph helper tests ---

func TestBuildNode(t *testing.T) {
	sessID := "sess-1"
	agent := "claude"
	model := "gpt-4"
	tokP := 100
	tokC := 50
	cost := 0.05

	trace := capture.Trace{
		ID:               "trace-1",
		SessionID:        &sessID,
		Agent:            &agent,
		Model:            model,
		TokensPrompt:     &tokP,
		TokensCompletion: &tokC,
		Cost:             &cost,
	}

	node := buildNode(trace, "completion", 0.9)
	if node.ID == "" {
		t.Error("node ID should not be empty")
	}
	if node.TraceID != "trace-1" {
		t.Errorf("expected trace ID trace-1, got %s", node.TraceID)
	}
	if node.SessionID != "sess-1" {
		t.Errorf("expected session ID sess-1, got %s", node.SessionID)
	}
	if node.Type != "completion" {
		t.Errorf("expected type completion, got %s", node.Type)
	}
	if node.Tokens == nil || *node.Tokens != 150 {
		t.Errorf("expected tokens 150, got %v", node.Tokens)
	}
}

func TestBuildNodeNilSession(t *testing.T) {
	trace := capture.Trace{
		ID:    "trace-1",
		Model: "gpt-4",
	}
	node := buildNode(trace, "request", 1.0)
	if node.SessionID != "unknown" {
		t.Errorf("expected session ID 'unknown' for nil session, got %s", node.SessionID)
	}
}

func TestBuildEdge(t *testing.T) {
	edge := buildEdge("sess-1", "node-a", "node-b", "triggered", 0.9)
	if edge.ID == "" {
		t.Error("edge ID should not be empty")
	}
	if edge.SessionID != "sess-1" {
		t.Errorf("expected session ID sess-1, got %s", edge.SessionID)
	}
	if edge.FromNode != "node-a" {
		t.Errorf("expected from node-a, got %s", edge.FromNode)
	}
	if edge.ToNode != "node-b" {
		t.Errorf("expected to node-b, got %s", edge.ToNode)
	}
	if edge.Type != "triggered" {
		t.Errorf("expected type triggered, got %s", edge.Type)
	}
	if edge.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", edge.Confidence)
	}
}
