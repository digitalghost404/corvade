package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test.db")
}

func TestNewStore(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	if s.DB() == nil {
		t.Fatal("DB() returned nil")
	}

	// Verify tables exist
	tables := []string{"traces", "sessions", "graph_nodes", "graph_edges"}
	for _, table := range tables {
		var name string
		err := s.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestNewStoreInvalidPath(t *testing.T) {
	_, err := NewStore("/nonexistent/dir/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestInsertAndGetTrace(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	sessID := "sess-1"
	agent := "claude"
	step := "plan"
	resp := `{"choices":[]}`
	tokP := 100
	tokC := 50
	tokCached := 10
	cost := 0.05
	latency := 1200
	ttft := 200
	keyHash := "abc123"

	trace := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{"prompt":"hello"}`,
		Response:         &resp,
		StatusCode:       200,
		SessionID:        &sessID,
		Agent:            &agent,
		Step:             &step,
		TokensPrompt:     &tokP,
		TokensCompletion: &tokC,
		TokensCached:     &tokCached,
		Cost:             &cost,
		LatencyMS:        &latency,
		TTFTMS:           &ttft,
		APIKeyHash:       &keyHash,
	}

	id, err := s.InsertTrace(trace)
	if err != nil {
		t.Fatalf("InsertTrace failed: %v", err)
	}
	if id == "" {
		t.Fatal("InsertTrace returned empty ID")
	}

	got, err := s.GetTrace(id)
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID: got %s, want %s", got.ID, id)
	}
	if got.Provider != "openai" {
		t.Errorf("Provider: got %s, want openai", got.Provider)
	}
	if got.Model != "gpt-4" {
		t.Errorf("Model: got %s, want gpt-4", got.Model)
	}
	if got.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", got.StatusCode)
	}
	if got.SessionID == nil || *got.SessionID != sessID {
		t.Errorf("SessionID: got %v, want %s", got.SessionID, sessID)
	}
	if got.Agent == nil || *got.Agent != agent {
		t.Errorf("Agent: got %v, want %s", got.Agent, agent)
	}
	if got.TokensPrompt == nil || *got.TokensPrompt != tokP {
		t.Errorf("TokensPrompt: got %v, want %d", got.TokensPrompt, tokP)
	}
	if got.Cost == nil || *got.Cost != cost {
		t.Errorf("Cost: got %v, want %f", got.Cost, cost)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestGetTraceNotFound(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	_, err = s.GetTrace("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent trace")
	}
}

func TestListTraces(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert multiple traces with different providers/models
	for i := 0; i < 5; i++ {
		provider := "openai"
		model := "gpt-4"
		if i%2 == 0 {
			provider = "anthropic"
			model = "claude-3"
		}
		sessID := fmt.Sprintf("sess-%d", i%2)
		tr := Trace{
			Provider:   provider,
			Model:      model,
			Request:    `{}`,
			StatusCode: 200,
			SessionID:  &sessID,
		}
		_, err := s.InsertTrace(tr)
		if err != nil {
			t.Fatalf("InsertTrace[%d] failed: %v", i, err)
		}
	}

	// List all
	all, err := s.ListTraces(TraceFilter{})
	if err != nil {
		t.Fatalf("ListTraces (all) failed: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("ListTraces (all): got %d, want 5", len(all))
	}

	// Filter by provider
	prov := "anthropic"
	filtered, err := s.ListTraces(TraceFilter{Provider: &prov})
	if err != nil {
		t.Fatalf("ListTraces (provider) failed: %v", err)
	}
	if len(filtered) != 3 {
		t.Errorf("ListTraces (provider=anthropic): got %d, want 3", len(filtered))
	}

	// Filter by model
	mod := "gpt-4"
	filtered, err = s.ListTraces(TraceFilter{Model: &mod})
	if err != nil {
		t.Fatalf("ListTraces (model) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListTraces (model=gpt-4): got %d, want 2", len(filtered))
	}

	// Filter by session
	sessFilter := "sess-0"
	filtered, err = s.ListTraces(TraceFilter{SessionID: &sessFilter})
	if err != nil {
		t.Fatalf("ListTraces (session) failed: %v", err)
	}
	if len(filtered) != 3 {
		t.Errorf("ListTraces (session=sess-0): got %d, want 3", len(filtered))
	}

	// Filter with limit
	limit := 2
	filtered, err = s.ListTraces(TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("ListTraces (limit) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListTraces (limit=2): got %d, want 2", len(filtered))
	}
}

func TestInsertAndGetSession(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	agent := "claude"
	endTime := time.Now().UTC().Add(time.Hour)

	sess := Session{
		Agent:     &agent,
		StartTime: time.Now().UTC(),
		EndTime:   &endTime,
		Status:    "active",
	}

	id, err := s.InsertSession(sess)
	if err != nil {
		t.Fatalf("InsertSession failed: %v", err)
	}
	if id == "" {
		t.Fatal("InsertSession returned empty ID")
	}

	got, err := s.GetSession(id)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID: got %s, want %s", got.ID, id)
	}
	if got.Agent == nil || *got.Agent != agent {
		t.Errorf("Agent: got %v, want %s", got.Agent, agent)
	}
	if got.Status != "active" {
		t.Errorf("Status: got %s, want active", got.Status)
	}
	if got.EndTime == nil {
		t.Error("EndTime should not be nil")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestInsertSessionNilEndTime(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	sess := Session{
		StartTime: time.Now().UTC(),
		Status:    "active",
	}

	id, err := s.InsertSession(sess)
	if err != nil {
		t.Fatalf("InsertSession failed: %v", err)
	}

	got, err := s.GetSession(id)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.EndTime != nil {
		t.Errorf("EndTime: got %v, want nil", got.EndTime)
	}
}

func TestListSessions(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	for i := 0; i < 3; i++ {
		sess := Session{
			StartTime: time.Now().UTC(),
			Status:    "active",
		}
		_, err := s.InsertSession(sess)
		if err != nil {
			t.Fatalf("InsertSession[%d] failed: %v", i, err)
		}
	}

	list, err := s.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("ListSessions: got %d, want 3", len(list))
	}
}

func TestUpdateSessionStats(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	sess := Session{
		StartTime: time.Now().UTC(),
		Status:    "active",
	}
	sessID, err := s.InsertSession(sess)
	if err != nil {
		t.Fatalf("InsertSession failed: %v", err)
	}

	// Insert traces for the session
	tokP := 100
	tokC := 50
	cost := 0.05
	for i := 0; i < 3; i++ {
		tr := Trace{
			Provider:         "openai",
			Model:            "gpt-4",
			Request:          `{}`,
			StatusCode:       200,
			SessionID:        &sessID,
			TokensPrompt:     &tokP,
			TokensCompletion: &tokC,
			Cost:             &cost,
		}
		_, err := s.InsertTrace(tr)
		if err != nil {
			t.Fatalf("InsertTrace[%d] failed: %v", i, err)
		}
	}

	err = s.UpdateSessionStats(sessID)
	if err != nil {
		t.Fatalf("UpdateSessionStats failed: %v", err)
	}

	got, err := s.GetSession(sessID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.TraceCount != 3 {
		t.Errorf("TraceCount: got %d, want 3", got.TraceCount)
	}
	// total tokens = 3 * (100 + 50) = 450
	if got.TotalTokens != 450 {
		t.Errorf("TotalTokens: got %d, want 450", got.TotalTokens)
	}
	// total cost = 3 * 0.05 = 0.15
	if got.TotalCost < 0.14 || got.TotalCost > 0.16 {
		t.Errorf("TotalCost: got %f, want ~0.15", got.TotalCost)
	}
}

func TestGraphNodeAndEdge(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Create a session first
	sess := Session{StartTime: time.Now().UTC(), Status: "active"}
	sessID, err := s.InsertSession(sess)
	if err != nil {
		t.Fatalf("InsertSession failed: %v", err)
	}

	// Create traces
	tr := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200, SessionID: &sessID}
	trID1, _ := s.InsertTrace(tr)
	trID2, _ := s.InsertTrace(tr)

	// Insert nodes
	model := "gpt-4"
	node1 := GraphNode{
		TraceID:    trID1,
		SessionID:  sessID,
		Type:       "llm_call",
		Model:      &model,
		Confidence: 0.95,
	}
	node2 := GraphNode{
		TraceID:    trID2,
		SessionID:  sessID,
		Type:       "llm_call",
		Model:      &model,
		Confidence: 0.90,
	}

	nID1, err := s.InsertGraphNode(node1)
	if err != nil {
		t.Fatalf("InsertGraphNode failed: %v", err)
	}
	nID2, err := s.InsertGraphNode(node2)
	if err != nil {
		t.Fatalf("InsertGraphNode failed: %v", err)
	}

	// Insert edge
	edge := GraphEdge{
		SessionID:  sessID,
		FromNode:   nID1,
		ToNode:     nID2,
		Type:       "sequence",
		Confidence: 0.85,
	}
	eID, err := s.InsertGraphEdge(edge)
	if err != nil {
		t.Fatalf("InsertGraphEdge failed: %v", err)
	}
	if eID == "" {
		t.Fatal("InsertGraphEdge returned empty ID")
	}

	// Get session graph
	nodes, edges, err := s.GetSessionGraph(sessID)
	if err != nil {
		t.Fatalf("GetSessionGraph failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("Nodes: got %d, want 2", len(nodes))
	}
	if len(edges) != 1 {
		t.Errorf("Edges: got %d, want 1", len(edges))
	}
	if edges[0].Type != "sequence" {
		t.Errorf("Edge type: got %s, want sequence", edges[0].Type)
	}
}

func TestStoreReopensPersisted(t *testing.T) {
	dbPath := tempDB(t)

	// Create store and insert data
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	tr := Trace{Provider: "test", Model: "test", Request: `{}`, StatusCode: 200}
	id, _ := s.InsertTrace(tr)
	s.Close()

	// Reopen and verify data persists
	s2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore (reopen) failed: %v", err)
	}
	defer s2.Close()

	got, err := s2.GetTrace(id)
	if err != nil {
		t.Fatalf("GetTrace after reopen failed: %v", err)
	}
	if got.Provider != "test" {
		t.Errorf("Provider after reopen: got %s, want test", got.Provider)
	}
}

func TestNewStoreCreatesDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore should create parent dirs: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("parent directory was not created")
	}
}
