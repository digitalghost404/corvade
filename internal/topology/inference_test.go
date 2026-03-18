package topology

import (
	"encoding/json"
	"os"
	"testing"

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
}
