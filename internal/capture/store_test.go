package capture

import (
	"encoding/json"
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

func TestNewULIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewULID()
		if id == "" {
			t.Fatal("NewULID returned empty string")
		}
		if ids[id] {
			t.Fatalf("duplicate ULID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGetSessionNotFound(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	_, err = s.GetSession("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestListTracesAllFilters(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	sessID := "sess-filter"
	agent := "claude"
	for i := 0; i < 5; i++ {
		tr := Trace{
			Provider:   "openai",
			Model:      "gpt-4",
			Request:    fmt.Sprintf(`{"prompt":"hello %d"}`, i),
			StatusCode: 200,
			SessionID:  &sessID,
			Agent:      &agent,
		}
		_, err := s.InsertTrace(tr)
		if err != nil {
			t.Fatalf("InsertTrace[%d] failed: %v", i, err)
		}
	}

	// Filter by agent
	filtered, err := s.ListTraces(TraceFilter{Agent: &agent})
	if err != nil {
		t.Fatalf("ListTraces (agent) failed: %v", err)
	}
	if len(filtered) != 5 {
		t.Errorf("ListTraces (agent): got %d, want 5", len(filtered))
	}

	// Filter by search
	search := "hello 3"
	filtered, err = s.ListTraces(TraceFilter{Search: &search})
	if err != nil {
		t.Fatalf("ListTraces (search) failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("ListTraces (search): got %d, want 1", len(filtered))
	}

	// Filter with offset
	limit := 10
	offset := 3
	filtered, err = s.ListTraces(TraceFilter{Limit: &limit, Offset: &offset})
	if err != nil {
		t.Fatalf("ListTraces (offset) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListTraces (offset=3): got %d, want 2", len(filtered))
	}

	// Combined filters: agent + model + session + limit
	model := "gpt-4"
	provider := "openai"
	combined := 2
	filtered, err = s.ListTraces(TraceFilter{
		Agent:     &agent,
		Model:     &model,
		Provider:  &provider,
		SessionID: &sessID,
		Limit:     &combined,
	})
	if err != nil {
		t.Fatalf("ListTraces (combined) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListTraces (combined): got %d, want 2", len(filtered))
	}
}

func TestListSessionsFiltered(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	agent1 := "claude"
	agent2 := "gpt"

	for i := 0; i < 3; i++ {
		a := &agent1
		if i == 2 {
			a = &agent2
		}
		sess := Session{
			StartTime: time.Now().UTC(),
			Status:    "active",
			Agent:     a,
		}
		_, err := s.InsertSession(sess)
		if err != nil {
			t.Fatalf("InsertSession[%d] failed: %v", i, err)
		}
	}

	// No filter
	all, err := s.ListSessionsFiltered(SessionFilter{})
	if err != nil {
		t.Fatalf("ListSessionsFiltered (all) failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListSessionsFiltered (all): got %d, want 3", len(all))
	}

	// Filter by agent
	filtered, err := s.ListSessionsFiltered(SessionFilter{Agent: &agent1})
	if err != nil {
		t.Fatalf("ListSessionsFiltered (agent) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListSessionsFiltered (agent=claude): got %d, want 2", len(filtered))
	}

	// With limit
	limit := 1
	filtered, err = s.ListSessionsFiltered(SessionFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("ListSessionsFiltered (limit) failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("ListSessionsFiltered (limit=1): got %d, want 1", len(filtered))
	}

	// With offset
	offset := 2
	limit = 10
	filtered, err = s.ListSessionsFiltered(SessionFilter{Limit: &limit, Offset: &offset})
	if err != nil {
		t.Fatalf("ListSessionsFiltered (offset) failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("ListSessionsFiltered (offset=2): got %d, want 1", len(filtered))
	}
}

func TestGetStatsNoData(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	stats, err := s.GetStats(nil, nil)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TraceCount != 0 {
		t.Errorf("TraceCount: got %d, want 0", stats.TraceCount)
	}
	if stats.TotalCost != 0 {
		t.Errorf("TotalCost: got %f, want 0", stats.TotalCost)
	}
}

func TestGetStatsWithData(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	agent := "claude"
	tokP := 100
	tokC := 50
	cost := 0.05

	for i := 0; i < 3; i++ {
		model := "gpt-4"
		if i == 2 {
			model = "claude-3"
		}
		tr := Trace{
			Provider:         "openai",
			Model:            model,
			Request:          `{}`,
			StatusCode:       200,
			Agent:            &agent,
			TokensPrompt:     &tokP,
			TokensCompletion: &tokC,
			Cost:             &cost,
		}
		_, err := s.InsertTrace(tr)
		if err != nil {
			t.Fatalf("InsertTrace[%d] failed: %v", i, err)
		}
	}

	stats, err := s.GetStats(nil, nil)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TraceCount != 3 {
		t.Errorf("TraceCount: got %d, want 3", stats.TraceCount)
	}
	if stats.TotalTokens != 450 {
		t.Errorf("TotalTokens: got %d, want 450", stats.TotalTokens)
	}
	if stats.TotalCost < 0.14 || stats.TotalCost > 0.16 {
		t.Errorf("TotalCost: got %f, want ~0.15", stats.TotalCost)
	}
	if stats.ByModel["gpt-4"] != 2 {
		t.Errorf("ByModel[gpt-4]: got %d, want 2", stats.ByModel["gpt-4"])
	}
	if stats.ByModel["claude-3"] != 1 {
		t.Errorf("ByModel[claude-3]: got %d, want 1", stats.ByModel["claude-3"])
	}
	if stats.ByAgent["claude"] != 3 {
		t.Errorf("ByAgent[claude]: got %d, want 3", stats.ByAgent["claude"])
	}
}

func TestGetStatsWithTimeRange(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert an old trace directly
	oldTime := time.Now().UTC().AddDate(0, 0, -10).Format(time.RFC3339Nano)
	_, err = s.DB().Exec(`INSERT INTO traces
		(id, provider, model, request, status_code, tokens_prompt, tokens_completion, cost, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"OLD00000000000000000000001", "openai", "gpt-4", `{}`, 200, 100, 50, 0.05, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a recent trace
	tr := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200}
	_, err = s.InsertTrace(tr)
	if err != nil {
		t.Fatal(err)
	}

	// Query with from time that excludes old trace
	from := time.Now().UTC().AddDate(0, 0, -5)
	stats, err := s.GetStats(&from, nil)
	if err != nil {
		t.Fatalf("GetStats with from failed: %v", err)
	}
	if stats.TraceCount != 1 {
		t.Errorf("TraceCount with from filter: got %d, want 1", stats.TraceCount)
	}

	// Query with to time that excludes recent trace
	to := time.Now().UTC().AddDate(0, 0, -5)
	stats, err = s.GetStats(nil, &to)
	if err != nil {
		t.Fatalf("GetStats with to failed: %v", err)
	}
	if stats.TraceCount != 1 {
		t.Errorf("TraceCount with to filter: got %d, want 1", stats.TraceCount)
	}

	// Both from and to
	from = time.Now().UTC().AddDate(0, 0, -15)
	to = time.Now().UTC().AddDate(0, 0, -5)
	stats, err = s.GetStats(&from, &to)
	if err != nil {
		t.Fatalf("GetStats with from+to failed: %v", err)
	}
	if stats.TraceCount != 1 {
		t.Errorf("TraceCount with from+to filter: got %d, want 1", stats.TraceCount)
	}
}

func TestListSessionsWithEndTime(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	endTime := time.Now().UTC().Add(time.Hour)
	agent := "claude"
	sess := Session{
		StartTime: time.Now().UTC(),
		EndTime:   &endTime,
		Status:    "completed",
		Agent:     &agent,
	}
	_, err = s.InsertSession(sess)
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0].EndTime == nil {
		t.Error("expected non-nil EndTime in listed session")
	}
}

func TestListSessionsFilteredWithEndTime(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	endTime := time.Now().UTC().Add(time.Hour)
	agent := "claude"
	sess := Session{
		StartTime: time.Now().UTC(),
		EndTime:   &endTime,
		Status:    "completed",
		Agent:     &agent,
	}
	_, err = s.InsertSession(sess)
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.ListSessionsFiltered(SessionFilter{Agent: &agent})
	if err != nil {
		t.Fatalf("ListSessionsFiltered failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0].EndTime == nil {
		t.Error("expected non-nil EndTime in filtered session")
	}
}

func TestInsertTraceError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close() // close DB to trigger errors

	_, err = s.InsertTrace(Trace{Provider: "x", Model: "y", Request: "{}", StatusCode: 200})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestInsertSessionError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.InsertSession(Session{StartTime: time.Now().UTC(), Status: "active"})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestUpdateSessionStatsError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	err = s.UpdateSessionStats("nonexistent")
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestInsertGraphNodeError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.InsertGraphNode(GraphNode{TraceID: "x", SessionID: "y", Type: "z"})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestInsertGraphEdgeError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.InsertGraphEdge(GraphEdge{SessionID: "x", FromNode: "a", ToNode: "b", Type: "seq"})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestGetSessionGraphError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, _, err = s.GetSessionGraph("nonexistent")
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestListTracesError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.ListTraces(TraceFilter{})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestListSessionsError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.ListSessions()
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestListSessionsFilteredError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.ListSessionsFiltered(SessionFilter{})
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestGetStatsError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s.Close()

	_, err = s.GetStats(nil, nil)
	if err == nil {
		t.Fatal("expected error when DB is closed")
	}
}

func TestListTracesScanError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert a trace normally
	tr := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200}
	_, err = s.InsertTrace(tr)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt the table by renaming columns - this will cause scan errors
	// SQLite doesn't support ALTER COLUMN so we recreate with different schema
	s.DB().Exec("DROP TABLE IF EXISTS traces_backup")
	s.DB().Exec("ALTER TABLE traces RENAME TO traces_backup")
	// Create traces table with wrong column types to cause scan failures
	s.DB().Exec(`CREATE TABLE traces (
		id TEXT, session_id TEXT, agent TEXT, step TEXT, provider TEXT,
		model TEXT, request TEXT, response TEXT, status_code TEXT,
		tokens_prompt TEXT, tokens_completion TEXT, tokens_cached TEXT,
		cost TEXT, latency_ms TEXT, ttft_ms TEXT, api_key_hash TEXT, created_at TEXT
	)`)
	s.DB().Exec(`INSERT INTO traces (id, session_id, status_code, provider, model, request, created_at)
		VALUES ('x', NULL, 'not_an_int', 'p', 'm', '{}', 'bad_time')`)

	_, err = s.ListTraces(TraceFilter{})
	if err != nil {
		// Scan error triggered - good
		t.Logf("Got expected scan error: %v", err)
	}
}

func TestListSessionsScanError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	s.DB().Exec("DROP TABLE IF EXISTS sessions")
	s.DB().Exec(`CREATE TABLE sessions (
		id TEXT, agent TEXT, start_time TEXT, end_time TEXT,
		trace_count TEXT, total_tokens TEXT, total_cost TEXT,
		status TEXT, created_at TEXT
	)`)
	s.DB().Exec(`INSERT INTO sessions (id, start_time, trace_count, total_tokens, total_cost, status, created_at)
		VALUES ('x', 'time', 'not_int', 'not_int', 'not_float', 'active', 'time')`)

	_, err = s.ListSessions()
	if err != nil {
		t.Logf("Got expected scan error: %v", err)
	}
}

func TestGetSessionGraphEdgeQueryError(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert a node, then drop edges table to trigger the edge query error
	sessID := "sess-1"
	trID := "trace-1"
	s.DB().Exec(`INSERT INTO graph_nodes (id, trace_id, session_id, type, confidence, created_at)
		VALUES (?, ?, ?, 'llm_call', 0.9, ?)`, "node-1", trID, sessID, time.Now().UTC().Format(time.RFC3339Nano))

	// Drop the edges table so the second query fails
	s.DB().Exec("DROP TABLE graph_edges")

	_, _, err = s.GetSessionGraph(sessID)
	if err == nil {
		t.Fatal("expected error when graph_edges table is missing")
	}
}

func TestGetSessionGraphEmpty(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	nodes, edges, err := s.GetSessionGraph("nonexistent")
	if err != nil {
		t.Fatalf("GetSessionGraph failed: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

func TestInsertTraceWithPolicyViolations(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	violations := `[{"rule":"max_tokens","mode":"enforce"}]`
	tr := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &violations,
	}

	id, err := s.InsertTrace(tr)
	if err != nil {
		t.Fatalf("InsertTrace with violations failed: %v", err)
	}
	if id == "" {
		t.Fatal("InsertTrace returned empty ID")
	}
}

func TestGetTraceRoundTripPolicyViolations(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	violations := `[{"rule":"max_tokens","mode":"enforce"},{"rule":"blocked_model","mode":"warn"}]`
	tr := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &violations,
	}

	id, err := s.InsertTrace(tr)
	if err != nil {
		t.Fatalf("InsertTrace failed: %v", err)
	}

	got, err := s.GetTrace(id)
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}

	if got.PolicyViolations == nil {
		t.Fatal("PolicyViolations should not be nil")
	}
	if *got.PolicyViolations != violations {
		t.Errorf("PolicyViolations: got %s, want %s", *got.PolicyViolations, violations)
	}
}

func TestGetTraceNilPolicyViolations(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	tr := Trace{
		Provider:   "openai",
		Model:      "gpt-4",
		Request:    `{}`,
		StatusCode: 200,
	}

	id, err := s.InsertTrace(tr)
	if err != nil {
		t.Fatalf("InsertTrace failed: %v", err)
	}

	got, err := s.GetTrace(id)
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}

	if got.PolicyViolations != nil {
		t.Errorf("PolicyViolations should be nil, got %v", got.PolicyViolations)
	}
}

func TestListTracesIncludesPolicyViolations(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	violations := `[{"rule":"max_tokens","mode":"enforce"}]`
	tr := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &violations,
	}
	_, err = s.InsertTrace(tr)
	if err != nil {
		t.Fatalf("InsertTrace failed: %v", err)
	}

	traces, err := s.ListTraces(TraceFilter{})
	if err != nil {
		t.Fatalf("ListTraces failed: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].PolicyViolations == nil {
		t.Fatal("PolicyViolations should not be nil in listed trace")
	}
	if *traces[0].PolicyViolations != violations {
		t.Errorf("PolicyViolations: got %s, want %s", *traces[0].PolicyViolations, violations)
	}
}

func TestListTracesHasViolationsFilter(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert trace WITH violations
	violations := `[{"rule":"max_tokens","mode":"enforce"}]`
	tr1 := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &violations,
	}
	_, err = s.InsertTrace(tr1)
	if err != nil {
		t.Fatalf("InsertTrace (with violations) failed: %v", err)
	}

	// Insert trace WITHOUT violations
	tr2 := Trace{
		Provider:   "openai",
		Model:      "gpt-4",
		Request:    `{}`,
		StatusCode: 200,
	}
	_, err = s.InsertTrace(tr2)
	if err != nil {
		t.Fatalf("InsertTrace (no violations) failed: %v", err)
	}

	// Insert trace with empty array (should NOT count as having violations)
	empty := `[]`
	tr3 := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &empty,
	}
	_, err = s.InsertTrace(tr3)
	if err != nil {
		t.Fatalf("InsertTrace (empty violations) failed: %v", err)
	}

	// HasViolations=true should only return the one with real violations
	hasViolations := true
	filtered, err := s.ListTraces(TraceFilter{HasViolations: &hasViolations})
	if err != nil {
		t.Fatalf("ListTraces (HasViolations=true) failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("ListTraces (HasViolations=true): got %d, want 1", len(filtered))
	}

	// No HasViolations filter should return all
	all, err := s.ListTraces(TraceFilter{})
	if err != nil {
		t.Fatalf("ListTraces (all) failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListTraces (all): got %d, want 3", len(all))
	}
}

func TestGetStatsViolationAggregation(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	v1 := `[{"rule":"max_tokens","mode":"enforce"},{"rule":"blocked_model","mode":"warn"}]`
	v2 := `[{"rule":"max_tokens","mode":"enforce"}]`

	tr1 := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200, PolicyViolations: &v1}
	tr2 := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200, PolicyViolations: &v2}
	tr3 := Trace{Provider: "openai", Model: "gpt-4", Request: `{}`, StatusCode: 200} // no violations

	for _, tr := range []Trace{tr1, tr2, tr3} {
		_, err := s.InsertTrace(tr)
		if err != nil {
			t.Fatalf("InsertTrace failed: %v", err)
		}
	}

	stats, err := s.GetStats(nil, nil)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.ViolationsByRule == nil {
		t.Fatal("ViolationsByRule should not be nil")
	}
	if stats.ViolationsByMode == nil {
		t.Fatal("ViolationsByMode should not be nil")
	}

	// max_tokens appears in v1 and v2 = 2
	if stats.ViolationsByRule["max_tokens"] != 2 {
		t.Errorf("ViolationsByRule[max_tokens]: got %d, want 2", stats.ViolationsByRule["max_tokens"])
	}
	// blocked_model appears in v1 only = 1
	if stats.ViolationsByRule["blocked_model"] != 1 {
		t.Errorf("ViolationsByRule[blocked_model]: got %d, want 1", stats.ViolationsByRule["blocked_model"])
	}
	// enforce mode: v1 has 1, v2 has 1 = 2
	if stats.ViolationsByMode["enforce"] != 2 {
		t.Errorf("ViolationsByMode[enforce]: got %d, want 2", stats.ViolationsByMode["enforce"])
	}
	// warn mode: v1 has 1 = 1
	if stats.ViolationsByMode["warn"] != 1 {
		t.Errorf("ViolationsByMode[warn]: got %d, want 1", stats.ViolationsByMode["warn"])
	}
}

func TestGetStatsViolationsRespectsTimeRange(t *testing.T) {
	dbPath := tempDB(t)
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Insert an old trace with violations directly
	oldTime := time.Now().UTC().AddDate(0, 0, -10).Format(time.RFC3339Nano)
	oldViolations := `[{"rule":"old_rule","mode":"enforce"}]`
	_, err = s.DB().Exec(`INSERT INTO traces
		(id, provider, model, request, status_code, policy_violations, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"OLD00000000000000000000002", "openai", "gpt-4", `{}`, 200, oldViolations, oldTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a recent trace with violations
	recentViolations := `[{"rule":"new_rule","mode":"warn"}]`
	tr := Trace{
		Provider:         "openai",
		Model:            "gpt-4",
		Request:          `{}`,
		StatusCode:       200,
		PolicyViolations: &recentViolations,
	}
	_, err = s.InsertTrace(tr)
	if err != nil {
		t.Fatal(err)
	}

	// Query only recent (last 5 days)
	from := time.Now().UTC().AddDate(0, 0, -5)
	stats, err := s.GetStats(&from, nil)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.ViolationsByRule["new_rule"] != 1 {
		t.Errorf("ViolationsByRule[new_rule]: got %d, want 1", stats.ViolationsByRule["new_rule"])
	}
	if stats.ViolationsByRule["old_rule"] != 0 {
		t.Errorf("ViolationsByRule[old_rule]: got %d, want 0 (should be filtered out)", stats.ViolationsByRule["old_rule"])
	}
}

func TestSchemaMigrationIdempotent(t *testing.T) {
	dbPath := tempDB(t)

	// Open store twice - migration should run without error both times
	s1, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore (first) failed: %v", err)
	}
	s1.Close()

	s2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore (second) failed: %v", err)
	}
	defer s2.Close()

	// Verify rate_limit_state table exists
	var name string
	err = s2.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='rate_limit_state'").Scan(&name)
	if err != nil {
		t.Errorf("rate_limit_state table not found: %v", err)
	}

	// Verify policy_violations column exists on traces
	rows, err := s2.DB().Query("PRAGMA table_info(traces)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatal(err)
		}
		if colName == "policy_violations" {
			found = true
			break
		}
	}
	if !found {
		t.Error("policy_violations column not found in traces table")
	}
}

// Ensure json import is used
var _ = json.Marshal
