# Corvade v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Corvade v1 MVP — a Go proxy that intercepts agent-to-LLM traffic, captures it to SQLite, infers topology graphs, and serves a Next.js dashboard with timeline, topology graph, session diff, and narrative views.

**Architecture:** Go reverse proxy on port 4400 intercepts and forwards LLM API calls, writing captures to SQLite. A separate Go API server on port 4401 serves REST endpoints and WebSocket events to an embedded Next.js dashboard. A topology inference engine derives directed graphs from captured traces on ingest.

**Tech Stack:** Go 1.22+, SQLite (mattn/go-sqlite3), Cobra (CLI), Next.js 14 (App Router), React Flow, Dagre, Tailwind CSS, TypeScript, ULID

**Spec:** `docs/superpowers/specs/2026-03-17-corvade-design.md` (in obsidian vault — copy to corvade repo)

---

## File Map

### Go Backend

| File | Responsibility |
|------|---------------|
| `cmd/corvade/main.go` | CLI entrypoint, Cobra root command |
| `internal/config/config.go` | YAML config loading, defaults, auto-create |
| `internal/capture/models.go` | Go structs for Trace, Session, GraphNode, GraphEdge |
| `internal/capture/store.go` | SQLite connection, schema migration, CRUD operations |
| `internal/capture/retention.go` | Automatic cleanup of old traces |
| `internal/cost/models.go` | Model pricing lookup table |
| `internal/cost/calculator.go` | Cost calculation from token counts |
| `internal/proxy/server.go` | HTTP server, route detection, provider dispatch |
| `internal/proxy/stream.go` | SSE streaming passthrough with TeeReader capture |
| `internal/proxy/providers/openai.go` | OpenAI request/response parsing, token extraction |
| `internal/proxy/providers/anthropic.go` | Anthropic request/response parsing, token extraction |
| `internal/proxy/middleware.go` | Capture middleware, header extraction, WebSocket emit |
| `internal/topology/inference.go` | Main inference orchestrator, runs all heuristics |
| `internal/topology/fingerprint.go` | Conversation fingerprinting (message history hashing) |
| `internal/topology/toolcall.go` | Tool call ID chain resolution |
| `internal/topology/graph.go` | Node/edge construction, session grouping |
| `internal/api/server.go` | HTTP server for dashboard API + WebSocket |
| `internal/api/routes.go` | Route registration |
| `internal/api/handlers/traces.go` | GET /api/traces, GET /api/traces/:id |
| `internal/api/handlers/sessions.go` | GET /api/sessions, GET /api/sessions/:id |
| `internal/api/handlers/topology.go` | GET /api/sessions/:id/graph, diff, narrative |
| `internal/api/handlers/export.go` | GET /api/sessions/:id/export, GET /api/stats |
| `internal/api/ws.go` | WebSocket hub, event broadcasting |
| `internal/cli/start.go` | `corvade start` command |
| `internal/cli/tail.go` | `corvade tail` command |
| `internal/cli/doctor.go` | `corvade doctor` command |
| `internal/cli/sessions.go` | `corvade sessions` command |
| `internal/cli/inspect.go` | `corvade inspect` command |
| `internal/cli/export.go` | `corvade export` command |

### Next.js Dashboard

| File | Responsibility |
|------|---------------|
| `dashboard/src/app/layout.tsx` | Root layout, global styles, WebSocket provider |
| `dashboard/src/app/page.tsx` | Timeline view (home) |
| `dashboard/src/app/session/[id]/page.tsx` | Session detail + topology graph |
| `dashboard/src/app/diff/page.tsx` | Session diff view |
| `dashboard/src/components/Timeline.tsx` | Trace list with filtering, live updates |
| `dashboard/src/components/DetailInspector.tsx` | Request/response viewer, token breakdown |
| `dashboard/src/components/TopologyGraph.tsx` | React Flow DAG with custom nodes |
| `dashboard/src/components/NarrativeView.tsx` | Template narrative + LLM enhance button |
| `dashboard/src/components/SessionDiff.tsx` | Side-by-side diff visualization |
| `dashboard/src/components/TailStream.tsx` | Live WebSocket feed component |
| `dashboard/src/lib/api.ts` | REST API client |
| `dashboard/src/lib/ws.ts` | WebSocket client + React context |
| `dashboard/src/lib/narrative.ts` | Template-based narrative engine |
| `dashboard/src/lib/types.ts` | Auto-generated from Go structs |

### Test Fixtures

| File | Responsibility |
|------|---------------|
| `testdata/single-llm-call.json` | One OpenAI chat completion |
| `testdata/multi-turn-conversation.json` | 4-turn conversation thread |
| `testdata/tool-call-chain.json` | LLM → tool_call → tool_result → follow-up |
| `testdata/multi-agent-handoff.json` | Two agents with handoff |
| `testdata/error-retry-sequence.json` | Failed call + retry + success |

---

## Task 1: Project Scaffolding & Go Module Init

**Files:**
- Create: `go.mod`, `go.sum`, `cmd/corvade/main.go`, `Makefile`, `.gitignore`

- [ ] **Step 1: Initialize git repo**

```bash
cd ~/projects-wsl/corvade
git init
```

- [ ] **Step 2: Create .gitignore**

```
# Go
bin/
*.exe
*.test
*.out

# Dashboard
dashboard/node_modules/
dashboard/.next/
dashboard/out/

# Corvade data
*.db
*.db-wal
*.db-shm

# OS
.DS_Store
.env
```

- [ ] **Step 3: Initialize Go module**

```bash
go mod init github.com/corvade/corvade
```

- [ ] **Step 4: Create CLI entrypoint with Cobra**

Create `cmd/corvade/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "corvade",
	Short: "The AI agent control plane",
	Long:  "Corvade intercepts agent-to-LLM traffic for debugging, governance, and memory.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("corvade v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Install Cobra dependency**

```bash
go get github.com/spf13/cobra
go mod tidy
```

- [ ] **Step 6: Create Makefile**

```makefile
.PHONY: build test run clean

build:
	go build -o bin/corvade ./cmd/corvade

test:
	go test ./... -v

run:
	go run ./cmd/corvade

clean:
	rm -rf bin/
```

- [ ] **Step 7: Verify it builds and runs**

```bash
make build
./bin/corvade version
```

Expected: `corvade v0.1.0`

- [ ] **Step 8: Commit**

```bash
git add .
git commit -m "feat: scaffold project with Go module, Cobra CLI, and Makefile"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write config test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Port != 4400 {
		t.Errorf("expected port 4400, got %d", cfg.Port)
	}
	if cfg.DashboardPort != 4401 {
		t.Errorf("expected dashboard port 4401, got %d", cfg.DashboardPort)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.RetentionDays)
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected config file to be created")
	}
	if cfg.Port != 4400 {
		t.Errorf("expected default port 4400, got %d", cfg.Port)
	}

	// File should exist now
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file was not created on disk")
	}
}

func TestLoadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte("port: 5500\ndashboard_port: 5501\nretention_days: 7\n")
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, created, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not report created for existing file")
	}
	if cfg.Port != 5500 {
		t.Errorf("expected port 5500, got %d", cfg.Port)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("expected 7 retention days, got %d", cfg.RetentionDays)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/ -v
```

Expected: compilation error — package doesn't exist yet

- [ ] **Step 3: Implement config package**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type CostOverride struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
}

type Config struct {
	Port           int                      `yaml:"port"`
	DashboardPort  int                      `yaml:"dashboard_port"`
	RetentionDays  int                      `yaml:"retention_days"`
	TimingGapMS    int                      `yaml:"timing_gap_ms"`
	CostOverrides  map[string]CostOverride  `yaml:"cost_overrides,omitempty"`
}

func Default() Config {
	return Config{
		Port:          4400,
		DashboardPort: 4401,
		RetentionDays: 30,
		TimingGapMS:   2000,
	}
}

// Load reads config from path. If the file doesn't exist, creates it with defaults.
// Returns the config, whether the file was created, and any error.
func Load(path string) (Config, bool, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Create default config file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return cfg, false, fmt.Errorf("creating config directory: %w", err)
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return cfg, false, fmt.Errorf("marshaling default config: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return cfg, false, fmt.Errorf("writing default config: %w", err)
		}
		return cfg, true, nil
	}
	if err != nil {
		return cfg, false, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, false, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, false, nil
}

// DefaultPath returns the default config file path (~/.corvade/config.yaml)
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".corvade", "config.yaml")
}
```

- [ ] **Step 4: Install yaml dependency and run tests**

```bash
go get gopkg.in/yaml.v3
go test ./internal/config/ -v
```

Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config package with YAML loading and auto-creation"
```

---

## Task 3: Data Models & SQLite Store

**Files:**
- Create: `internal/capture/models.go`, `internal/capture/store.go`
- Test: `internal/capture/store_test.go`

- [ ] **Step 1: Create data models**

```go
package capture

import "time"

type Trace struct {
	ID              string    `json:"id"`
	SessionID       *string   `json:"session_id"`
	Agent           *string   `json:"agent"`
	Step            *string   `json:"step"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	Request         string    `json:"request"`
	Response        *string   `json:"response"`
	StatusCode      int       `json:"status_code"`
	TokensPrompt    *int      `json:"tokens_prompt"`
	TokensCompletion *int     `json:"tokens_completion"`
	TokensCached    *int      `json:"tokens_cached"`
	Cost            *float64  `json:"cost"`
	LatencyMS       *int      `json:"latency_ms"`
	TTFTMS          *int      `json:"ttft_ms"`
	APIKeyHash      *string   `json:"api_key_hash"`
	CreatedAt       time.Time `json:"created_at"`
}

type Session struct {
	ID          string    `json:"id"`
	Agent       *string   `json:"agent"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	TraceCount  int       `json:"trace_count"`
	TotalTokens int       `json:"total_tokens"`
	TotalCost   float64   `json:"total_cost"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type GraphNode struct {
	ID              string   `json:"id"`
	TraceID         string   `json:"trace_id"`
	SessionID       string   `json:"session_id"`
	Type            string   `json:"type"`
	Agent           *string  `json:"agent"`
	Step            *string  `json:"step"`
	Model           *string  `json:"model"`
	Tokens          *int     `json:"tokens"`
	Cost            *float64 `json:"cost"`
	LatencyMS       *int     `json:"latency_ms"`
	ContextSnapshot *string  `json:"context_snapshot"`
	Confidence      float64  `json:"confidence"`
	CreatedAt       time.Time `json:"created_at"`
}

type GraphEdge struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	FromNode   string    `json:"from_node"`
	ToNode     string    `json:"to_node"`
	Type       string    `json:"type"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Write store tests**

```go
package capture

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
}

func TestInsertAndGetTrace(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	trace := Trace{
		ID:         "01HTEST000000000000000001",
		Provider:   "openai",
		Model:      "gpt-4o",
		Request:    `{"model":"gpt-4o","messages":[]}`,
		StatusCode: 200,
		CreatedAt:  time.Now().UTC(),
	}

	if err := store.InsertTrace(&trace); err != nil {
		t.Fatalf("failed to insert trace: %v", err)
	}

	got, err := store.GetTrace(trace.ID)
	if err != nil {
		t.Fatalf("failed to get trace: %v", err)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", got.Model)
	}
	if got.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", got.Provider)
	}
}

func TestListTraces(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert 3 traces
	for i := 0; i < 3; i++ {
		agent := "test-agent"
		trace := Trace{
			ID:         fmt.Sprintf("01HTEST00000000000000000%d", i),
			Agent:      &agent,
			Provider:   "openai",
			Model:      "gpt-4o",
			Request:    `{}`,
			StatusCode: 200,
			CreatedAt:  time.Now().UTC(),
		}
		store.InsertTrace(&trace)
	}

	traces, err := store.ListTraces(TraceFilter{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}
	if len(traces) != 3 {
		t.Errorf("expected 3 traces, got %d", len(traces))
	}
}

func TestInsertAndGetSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	session := Session{
		ID:        "01HTEST000000000000000001",
		StartTime: time.Now().UTC(),
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	if err := store.InsertSession(&session); err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	got, err := store.GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/capture/ -v
```

Expected: compilation errors

- [ ] **Step 4: Implement store**

```go
package capture

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid/v2"
	"math/rand"
)

var entropy = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		agent       TEXT,
		start_time  TEXT NOT NULL,
		end_time    TEXT,
		trace_count INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		total_cost  REAL DEFAULT 0,
		status      TEXT DEFAULT 'active',
		created_at  TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS traces (
		id                TEXT PRIMARY KEY,
		session_id        TEXT,
		agent             TEXT,
		step              TEXT,
		provider          TEXT NOT NULL,
		model             TEXT NOT NULL,
		request           TEXT NOT NULL,
		response          TEXT,
		status_code       INTEGER NOT NULL,
		tokens_prompt     INTEGER,
		tokens_completion INTEGER,
		tokens_cached     INTEGER,
		cost              REAL,
		latency_ms        INTEGER,
		ttft_ms           INTEGER,
		api_key_hash      TEXT,
		created_at        TEXT NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS graph_nodes (
		id               TEXT PRIMARY KEY,
		trace_id         TEXT NOT NULL,
		session_id       TEXT NOT NULL,
		type             TEXT NOT NULL,
		agent            TEXT,
		step             TEXT,
		model            TEXT,
		tokens           INTEGER,
		cost             REAL,
		latency_ms       INTEGER,
		context_snapshot TEXT,
		confidence       REAL DEFAULT 1.0,
		created_at       TEXT NOT NULL,
		FOREIGN KEY (trace_id) REFERENCES traces(id),
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS graph_edges (
		id          TEXT PRIMARY KEY,
		session_id  TEXT NOT NULL,
		from_node   TEXT NOT NULL,
		to_node     TEXT NOT NULL,
		type        TEXT NOT NULL,
		confidence  REAL DEFAULT 1.0,
		created_at  TEXT NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id),
		FOREIGN KEY (from_node) REFERENCES graph_nodes(id),
		FOREIGN KEY (to_node) REFERENCES graph_nodes(id)
	);

	CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(session_id);
	CREATE INDEX IF NOT EXISTS idx_traces_agent ON traces(agent);
	CREATE INDEX IF NOT EXISTS idx_traces_created ON traces(created_at);
	CREATE INDEX IF NOT EXISTS idx_traces_model ON traces(model);
	CREATE INDEX IF NOT EXISTS idx_nodes_session ON graph_nodes(session_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_trace ON graph_nodes(trace_id);
	CREATE INDEX IF NOT EXISTS idx_edges_session ON graph_edges(session_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent);
	CREATE INDEX IF NOT EXISTS idx_sessions_start ON sessions(start_time);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *Store) InsertTrace(t *Trace) error {
	_, err := s.db.Exec(`
		INSERT INTO traces (id, session_id, agent, step, provider, model, request, response,
			status_code, tokens_prompt, tokens_completion, tokens_cached, cost, latency_ms,
			ttft_ms, api_key_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.SessionID, t.Agent, t.Step, t.Provider, t.Model, t.Request, t.Response,
		t.StatusCode, t.TokensPrompt, t.TokensCompletion, t.TokensCached, t.Cost,
		t.LatencyMS, t.TTFTMS, t.APIKeyHash, t.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetTrace(id string) (*Trace, error) {
	row := s.db.QueryRow(`SELECT id, session_id, agent, step, provider, model, request,
		response, status_code, tokens_prompt, tokens_completion, tokens_cached, cost,
		latency_ms, ttft_ms, api_key_hash, created_at FROM traces WHERE id = ?`, id)

	t := &Trace{}
	var createdAt string
	err := row.Scan(&t.ID, &t.SessionID, &t.Agent, &t.Step, &t.Provider, &t.Model,
		&t.Request, &t.Response, &t.StatusCode, &t.TokensPrompt, &t.TokensCompletion,
		&t.TokensCached, &t.Cost, &t.LatencyMS, &t.TTFTMS, &t.APIKeyHash, &createdAt)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return t, nil
}

type TraceFilter struct {
	Agent   *string
	Model   *string
	Status  *int
	From    *time.Time
	To      *time.Time
	Search  *string
	Limit   int
	Offset  int
}

func (s *Store) ListTraces(f TraceFilter) ([]Trace, error) {
	query := "SELECT id, session_id, agent, step, provider, model, request, response, status_code, tokens_prompt, tokens_completion, tokens_cached, cost, latency_ms, ttft_ms, api_key_hash, created_at FROM traces WHERE 1=1"
	args := []interface{}{}

	if f.Agent != nil {
		query += " AND agent = ?"
		args = append(args, *f.Agent)
	}
	if f.Model != nil {
		query += " AND model = ?"
		args = append(args, *f.Model)
	}
	if f.Status != nil {
		query += " AND status_code = ?"
		args = append(args, *f.Status)
	}
	if f.From != nil {
		query += " AND created_at >= ?"
		args = append(args, f.From.UTC().Format(time.RFC3339Nano))
	}
	if f.To != nil {
		query += " AND created_at <= ?"
		args = append(args, f.To.UTC().Format(time.RFC3339Nano))
	}
	if f.Search != nil {
		query += " AND (request LIKE ? OR response LIKE ?)"
		pattern := "%" + *f.Search + "%"
		args = append(args, pattern, pattern)
	}

	query += " ORDER BY created_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		var createdAt string
		err := rows.Scan(&t.ID, &t.SessionID, &t.Agent, &t.Step, &t.Provider, &t.Model,
			&t.Request, &t.Response, &t.StatusCode, &t.TokensPrompt, &t.TokensCompletion,
			&t.TokensCached, &t.Cost, &t.LatencyMS, &t.TTFTMS, &t.APIKeyHash, &createdAt)
		if err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		traces = append(traces, t)
	}
	return traces, nil
}

func (s *Store) InsertSession(sess *Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, agent, start_time, end_time, trace_count, total_tokens,
			total_cost, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.Agent, sess.StartTime.UTC().Format(time.RFC3339Nano), nil,
		sess.TraceCount, sess.TotalTokens, sess.TotalCost, sess.Status,
		sess.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, agent, start_time, end_time, trace_count, total_tokens,
		total_cost, status, created_at FROM sessions WHERE id = ?`, id)

	sess := &Session{}
	var startTime, createdAt string
	var endTime *string
	err := row.Scan(&sess.ID, &sess.Agent, &startTime, &endTime, &sess.TraceCount,
		&sess.TotalTokens, &sess.TotalCost, &sess.Status, &createdAt)
	if err != nil {
		return nil, err
	}
	sess.StartTime, _ = time.Parse(time.RFC3339Nano, startTime)
	sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if endTime != nil {
		t, _ := time.Parse(time.RFC3339Nano, *endTime)
		sess.EndTime = &t
	}
	return sess, nil
}

func (s *Store) ListSessions(agent *string, from, to *time.Time, limit, offset int) ([]Session, error) {
	query := "SELECT id, agent, start_time, end_time, trace_count, total_tokens, total_cost, status, created_at FROM sessions WHERE 1=1"
	args := []interface{}{}

	if agent != nil {
		query += " AND agent = ?"
		args = append(args, *agent)
	}
	if from != nil {
		query += " AND start_time >= ?"
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if to != nil {
		query += " AND start_time <= ?"
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}

	query += " ORDER BY start_time DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var startTime, createdAt string
		var endTime *string
		err := rows.Scan(&sess.ID, &sess.Agent, &startTime, &endTime, &sess.TraceCount,
			&sess.TotalTokens, &sess.TotalCost, &sess.Status, &createdAt)
		if err != nil {
			return nil, err
		}
		sess.StartTime, _ = time.Parse(time.RFC3339Nano, startTime)
		sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if endTime != nil {
			t, _ := time.Parse(time.RFC3339Nano, *endTime)
			sess.EndTime = &t
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *Store) UpdateSessionStats(sessionID string) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET
			trace_count = (SELECT COUNT(*) FROM traces WHERE session_id = ?),
			total_tokens = (SELECT COALESCE(SUM(COALESCE(tokens_prompt, 0) + COALESCE(tokens_completion, 0)), 0) FROM traces WHERE session_id = ?),
			total_cost = (SELECT COALESCE(SUM(cost), 0) FROM traces WHERE session_id = ?)
		WHERE id = ?`, sessionID, sessionID, sessionID, sessionID)
	return err
}

// Graph operations

func (s *Store) InsertGraphNode(n *GraphNode) error {
	_, err := s.db.Exec(`
		INSERT INTO graph_nodes (id, trace_id, session_id, type, agent, step, model,
			tokens, cost, latency_ms, context_snapshot, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.TraceID, n.SessionID, n.Type, n.Agent, n.Step, n.Model,
		n.Tokens, n.Cost, n.LatencyMS, n.ContextSnapshot, n.Confidence,
		n.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) InsertGraphEdge(e *GraphEdge) error {
	_, err := s.db.Exec(`
		INSERT INTO graph_edges (id, session_id, from_node, to_node, type, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.SessionID, e.FromNode, e.ToNode, e.Type, e.Confidence,
		e.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) GetSessionGraph(sessionID string) ([]GraphNode, []GraphEdge, error) {
	nodeRows, err := s.db.Query(`
		SELECT id, trace_id, session_id, type, agent, step, model, tokens, cost,
			latency_ms, context_snapshot, confidence, created_at
		FROM graph_nodes WHERE session_id = ? ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	defer nodeRows.Close()

	var nodes []GraphNode
	for nodeRows.Next() {
		var n GraphNode
		var createdAt string
		err := nodeRows.Scan(&n.ID, &n.TraceID, &n.SessionID, &n.Type, &n.Agent, &n.Step,
			&n.Model, &n.Tokens, &n.Cost, &n.LatencyMS, &n.ContextSnapshot, &n.Confidence,
			&createdAt)
		if err != nil {
			return nil, nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		nodes = append(nodes, n)
	}

	edgeRows, err := s.db.Query(`
		SELECT id, session_id, from_node, to_node, type, confidence, created_at
		FROM graph_edges WHERE session_id = ?`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	defer edgeRows.Close()

	var edges []GraphEdge
	for edgeRows.Next() {
		var e GraphEdge
		var createdAt string
		err := edgeRows.Scan(&e.ID, &e.SessionID, &e.FromNode, &e.ToNode, &e.Type,
			&e.Confidence, &createdAt)
		if err != nil {
			return nil, nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		edges = append(edges, e)
	}

	return nodes, edges, nil
}

// DB returns the underlying *sql.DB for use by retention and other packages
func (s *Store) DB() *sql.DB {
	return s.db
}
```

- [ ] **Step 5: Add missing import in test file**

Add `"fmt"` to the imports in `store_test.go`.

- [ ] **Step 6: Install dependencies and run tests**

```bash
go get github.com/mattn/go-sqlite3
go get github.com/oklog/ulid/v2
go test ./internal/capture/ -v
```

Expected: all tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/capture/ go.mod go.sum
git commit -m "feat: add data models and SQLite store with schema migration"
```

---

## Task 4: Cost Calculator

**Files:**
- Create: `internal/cost/models.go`, `internal/cost/calculator.go`
- Test: `internal/cost/calculator_test.go`

- [ ] **Step 1: Write cost calculator test**

```go
package cost

import "testing"

func TestCalculateCostOpenAI(t *testing.T) {
	tests := []struct {
		model            string
		promptTokens     int
		completionTokens int
		expectedCost     float64
	}{
		{"gpt-4o", 1000, 500, 0.0075},       // 1K * 0.0025 + 0.5K * 0.01
		{"gpt-4o-mini", 1000, 500, 0.00045},  // 1K * 0.00015 + 0.5K * 0.0006
		{"gpt-3.5-turbo", 1000, 500, 0.0015}, // 1K * 0.0005 + 0.5K * 0.002 (adjusted)
	}

	calc := NewCalculator(nil)
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cost := calc.Calculate(tt.model, tt.promptTokens, tt.completionTokens)
			if diff := cost - tt.expectedCost; diff > 0.0001 || diff < -0.0001 {
				t.Errorf("model %s: expected cost %.6f, got %.6f", tt.model, tt.expectedCost, cost)
			}
		})
	}
}

func TestCalculateCostAnthropic(t *testing.T) {
	calc := NewCalculator(nil)
	cost := calc.Calculate("claude-3-5-sonnet-20241022", 1000, 500)
	expected := 0.0105 // 1K * 0.003 + 0.5K * 0.015
	if diff := cost - expected; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("expected cost %.6f, got %.6f", expected, cost)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	calc := NewCalculator(nil)
	cost := calc.Calculate("unknown-model", 1000, 500)
	if cost != 0 {
		t.Errorf("expected 0 for unknown model, got %f", cost)
	}
}

func TestCalculateCostWithOverrides(t *testing.T) {
	overrides := map[string]ModelPricing{
		"gpt-4o": {InputPer1K: 0.001, OutputPer1K: 0.002},
	}
	calc := NewCalculator(overrides)
	cost := calc.Calculate("gpt-4o", 1000, 500)
	expected := 0.002 // 1K * 0.001 + 0.5K * 0.002
	if diff := cost - expected; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("expected cost %.6f, got %.6f", expected, cost)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/cost/ -v
```

- [ ] **Step 3: Implement cost models and calculator**

`internal/cost/models.go`:

```go
package cost

type ModelPricing struct {
	InputPer1K  float64
	OutputPer1K float64
}

// Default pricing table — updated with each Corvade release
var defaultPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":            {InputPer1K: 0.0025, OutputPer1K: 0.01},
	"gpt-4o-mini":       {InputPer1K: 0.00015, OutputPer1K: 0.0006},
	"gpt-4-turbo":       {InputPer1K: 0.01, OutputPer1K: 0.03},
	"gpt-3.5-turbo":     {InputPer1K: 0.0005, OutputPer1K: 0.002},
	"o1":                {InputPer1K: 0.015, OutputPer1K: 0.06},
	"o1-mini":           {InputPer1K: 0.003, OutputPer1K: 0.012},
	// Anthropic
	"claude-3-5-sonnet-20241022": {InputPer1K: 0.003, OutputPer1K: 0.015},
	"claude-3-5-haiku-20241022":  {InputPer1K: 0.0008, OutputPer1K: 0.004},
	"claude-3-opus-20240229":     {InputPer1K: 0.015, OutputPer1K: 0.075},
	"claude-sonnet-4-6":          {InputPer1K: 0.003, OutputPer1K: 0.015},
	"claude-haiku-4-5-20251001":  {InputPer1K: 0.0008, OutputPer1K: 0.004},
	"claude-opus-4-6":            {InputPer1K: 0.015, OutputPer1K: 0.075},
}
```

`internal/cost/calculator.go`:

```go
package cost

type Calculator struct {
	pricing map[string]ModelPricing
}

func NewCalculator(overrides map[string]ModelPricing) *Calculator {
	pricing := make(map[string]ModelPricing)
	for k, v := range defaultPricing {
		pricing[k] = v
	}
	for k, v := range overrides {
		pricing[k] = v
	}
	return &Calculator{pricing: pricing}
}

func (c *Calculator) Calculate(model string, promptTokens, completionTokens int) float64 {
	p, ok := c.pricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(promptTokens) / 1000.0 * p.InputPer1K
	outputCost := float64(completionTokens) / 1000.0 * p.OutputPer1K
	return inputCost + outputCost
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/cost/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cost/
git commit -m "feat: add cost calculator with model pricing table and user overrides"
```

---

## Task 5: Retention Cleanup

**Files:**
- Create: `internal/capture/retention.go`
- Test: `internal/capture/retention_test.go`

- [ ] **Step 1: Write retention test**

```go
package capture

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRetentionDeletesOldTraces(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Insert old trace (31 days ago)
	old := Trace{
		ID: "01HOLD0000000000000000001", Provider: "openai", Model: "gpt-4o",
		Request: `{}`, StatusCode: 200,
		CreatedAt: time.Now().UTC().AddDate(0, 0, -31),
	}
	store.InsertTrace(&old)

	// Insert recent trace
	recent := Trace{
		ID: "01HNEW0000000000000000001", Provider: "openai", Model: "gpt-4o",
		Request: `{}`, StatusCode: 200,
		CreatedAt: time.Now().UTC(),
	}
	store.InsertTrace(&recent)

	deleted, err := RunRetention(store, 30)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Recent trace should still exist
	traces, _ := store.ListTraces(TraceFilter{Limit: 10})
	if len(traces) != 1 {
		t.Errorf("expected 1 remaining trace, got %d", len(traces))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/capture/ -run TestRetention -v
```

- [ ] **Step 3: Implement retention**

```go
package capture

import (
	"fmt"
	"time"
)

// RunRetention deletes traces and associated graph data older than retentionDays.
// Returns the number of traces deleted.
func RunRetention(store *Store, retentionDays int) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format(time.RFC3339Nano)

	// Delete graph edges for old traces
	_, err := store.DB().Exec(`
		DELETE FROM graph_edges WHERE session_id IN (
			SELECT DISTINCT session_id FROM traces WHERE created_at < ?
		)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old graph edges: %w", err)
	}

	// Delete graph nodes for old traces
	_, err = store.DB().Exec(`DELETE FROM graph_nodes WHERE trace_id IN (
		SELECT id FROM traces WHERE created_at < ?
	)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old graph nodes: %w", err)
	}

	// Delete old traces
	result, err := store.DB().Exec("DELETE FROM traces WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old traces: %w", err)
	}

	deleted, _ := result.RowsAffected()

	// Clean up empty sessions
	_, err = store.DB().Exec(`DELETE FROM sessions WHERE id NOT IN (
		SELECT DISTINCT session_id FROM traces WHERE session_id IS NOT NULL
	)`)
	if err != nil {
		return deleted, fmt.Errorf("cleaning up empty sessions: %w", err)
	}

	return deleted, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/capture/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/capture/retention.go internal/capture/retention_test.go
git commit -m "feat: add retention cleanup for old traces and graph data"
```

---

## Task 6: OpenAI Provider Adapter

**Files:**
- Create: `internal/proxy/providers/openai.go`
- Test: `internal/proxy/providers/openai_test.go`

- [ ] **Step 1: Write OpenAI parser tests**

```go
package providers

import (
	"testing"
)

func TestOpenAIParseRequest(t *testing.T) {
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`

	info, err := OpenAIParseRequest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", info.Model)
	}
	if !info.Stream {
		t.Error("expected stream=true")
	}
}

func TestOpenAIParseResponse(t *testing.T) {
	body := `{"id":"chatcmpl-123","model":"gpt-4o","usage":{"prompt_tokens":10,"completion_tokens":20,"cached_tokens":5}}`

	info, err := OpenAIParseResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if info.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", info.PromptTokens)
	}
	if info.CompletionTokens != 20 {
		t.Errorf("expected 20 completion tokens, got %d", info.CompletionTokens)
	}
	if info.CachedTokens != 5 {
		t.Errorf("expected 5 cached tokens, got %d", info.CachedTokens)
	}
}

func TestOpenAIExtractToolCalls(t *testing.T) {
	body := `{"choices":[{"message":{"tool_calls":[{"id":"call_123","function":{"name":"search","arguments":"{\"q\":\"test\"}"}}]}}]}`

	calls := OpenAIExtractToolCalls([]byte(body))
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].ID != "call_123" {
		t.Errorf("expected call_123, got %s", calls[0].ID)
	}
	if calls[0].Name != "search" {
		t.Errorf("expected search, got %s", calls[0].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/proxy/providers/ -v
```

- [ ] **Step 3: Implement OpenAI provider**

```go
package providers

import "encoding/json"

type RequestInfo struct {
	Model  string
	Stream bool
}

type ResponseInfo struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

func OpenAIParseRequest(body []byte) (*RequestInfo, error) {
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestInfo{Model: req.Model, Stream: req.Stream}, nil
}

func OpenAIParseResponse(body []byte) (*ResponseInfo, error) {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			CachedTokens     int `json:"cached_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &ResponseInfo{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		CachedTokens:     resp.Usage.CachedTokens,
	}, nil
}

func OpenAIExtractToolCalls(body []byte) []ToolCall {
	var resp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Choices) == 0 {
		return nil
	}
	var calls []ToolCall
	for _, tc := range resp.Choices[0].Message.ToolCalls {
		calls = append(calls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return calls
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/proxy/providers/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/providers/openai.go internal/proxy/providers/openai_test.go
git commit -m "feat: add OpenAI provider adapter with request/response parsing"
```

---

## Task 7: Anthropic Provider Adapter

**Files:**
- Create: `internal/proxy/providers/anthropic.go`
- Test: `internal/proxy/providers/anthropic_test.go`

- [ ] **Step 1: Write Anthropic parser tests**

```go
package providers

import "testing"

func TestAnthropicParseRequest(t *testing.T) {
	body := `{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hello"}],"stream":true}`

	info, err := AnthropicParseRequest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected claude-3-5-sonnet-20241022, got %s", info.Model)
	}
	if !info.Stream {
		t.Error("expected stream=true")
	}
}

func TestAnthropicParseResponse(t *testing.T) {
	body := `{"id":"msg_123","model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":15,"output_tokens":25,"cache_read_input_tokens":3}}`

	info, err := AnthropicParseResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if info.PromptTokens != 15 {
		t.Errorf("expected 15 prompt tokens, got %d", info.PromptTokens)
	}
	if info.CompletionTokens != 25 {
		t.Errorf("expected 25 completion tokens, got %d", info.CompletionTokens)
	}
	if info.CachedTokens != 3 {
		t.Errorf("expected 3 cached tokens, got %d", info.CachedTokens)
	}
}

func TestAnthropicExtractToolUse(t *testing.T) {
	body := `{"content":[{"type":"tool_use","id":"toolu_123","name":"search","input":{"q":"test"}}]}`

	calls := AnthropicExtractToolCalls([]byte(body))
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].ID != "toolu_123" {
		t.Errorf("expected toolu_123, got %s", calls[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/proxy/providers/ -run TestAnthropic -v
```

- [ ] **Step 3: Implement Anthropic provider**

```go
package providers

import "encoding/json"

func AnthropicParseRequest(body []byte) (*RequestInfo, error) {
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestInfo{Model: req.Model, Stream: req.Stream}, nil
}

func AnthropicParseResponse(body []byte) (*ResponseInfo, error) {
	var resp struct {
		Usage struct {
			InputTokens          int `json:"input_tokens"`
			OutputTokens         int `json:"output_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &ResponseInfo{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		CachedTokens:     resp.Usage.CacheReadInputTokens,
	}, nil
}

func AnthropicExtractToolCalls(body []byte) []ToolCall {
	var resp struct {
		Content []struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	var calls []ToolCall
	for _, c := range resp.Content {
		if c.Type == "tool_use" {
			calls = append(calls, ToolCall{
				ID:        c.ID,
				Name:      c.Name,
				Arguments: string(c.Input),
			})
		}
	}
	return calls
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/proxy/providers/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/providers/anthropic.go internal/proxy/providers/anthropic_test.go
git commit -m "feat: add Anthropic provider adapter with request/response parsing"
```

---

## Task 8: Proxy Server (Core HTTP Reverse Proxy)

**Files:**
- Create: `internal/proxy/server.go`, `internal/proxy/stream.go`, `internal/proxy/middleware.go`
- Test: `internal/proxy/server_test.go`

- [ ] **Step 1: Write proxy integration test**

```go
package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
)

func TestProxyForwardsOpenAIRequest(t *testing.T) {
	// Mock upstream OpenAI API
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"chatcmpl-123","model":"gpt-4o","choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer upstream.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := capture.NewStore(dbPath)
	defer store.Close()
	calc := cost.NewCalculator(nil)

	srv := NewServer(store, calc, nil)
	srv.SetUpstreamURL("openai", upstream.URL)

	// Send request through proxy
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify response was forwarded
	respBody, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(respBody), "chatcmpl-123") {
		t.Error("response not forwarded correctly")
	}

	// Verify trace was captured
	traces, _ := store.ListTraces(capture.TraceFilter{Limit: 10})
	if len(traces) != 1 {
		t.Fatalf("expected 1 captured trace, got %d", len(traces))
	}
	if traces[0].Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", traces[0].Model)
	}
}

func TestProxyForwardsAnthropicRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"msg_123","model":"claude-3-5-sonnet-20241022","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":8,"output_tokens":3}}`))
	}))
	defer upstream.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := capture.NewStore(dbPath)
	defer store.Close()
	calc := cost.NewCalculator(nil)

	srv := NewServer(store, calc, nil)
	srv.SetUpstreamURL("anthropic", upstream.URL)

	body := `{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", "sk-ant-test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	traces, _ := store.ListTraces(capture.TraceFilter{Limit: 10})
	if len(traces) != 1 {
		t.Fatalf("expected 1 captured trace, got %d", len(traces))
	}
	if traces[0].Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", traces[0].Provider)
	}
}

func TestProxyExtractsCorvadeHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Corvade headers should NOT be forwarded upstream
		if r.Header.Get("X-Corvade-Agent") != "" {
			t.Error("X-Corvade-Agent should be stripped before forwarding")
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"chatcmpl-123","model":"gpt-4o","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := capture.NewStore(dbPath)
	defer store.Close()

	srv := NewServer(store, cost.NewCalculator(nil), nil)
	srv.SetUpstreamURL("openai", upstream.URL)

	body := `{"model":"gpt-4o","messages":[]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Corvade-Agent", "test-bot")
	req.Header.Set("X-Corvade-Session", "sess-001")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	traces, _ := store.ListTraces(capture.TraceFilter{Limit: 10})
	if len(traces) != 1 {
		t.Fatal("expected 1 trace")
	}
	if traces[0].Agent == nil || *traces[0].Agent != "test-bot" {
		t.Error("agent header not captured")
	}
	if traces[0].SessionID == nil || *traces[0].SessionID != "sess-001" {
		t.Error("session header not captured")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/proxy/ -v
```

- [ ] **Step 3: Implement proxy server**

`internal/proxy/server.go`:

```go
package proxy

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy/providers"
)

type EventEmitter interface {
	Emit(event string, data interface{})
}

type Server struct {
	store        *capture.Store
	calc         *cost.Calculator
	emitter      EventEmitter
	upstreamURLs map[string]string
	mux          *http.ServeMux
}

func NewServer(store *capture.Store, calc *cost.Calculator, emitter EventEmitter) *Server {
	s := &Server{
		store:   store,
		calc:    calc,
		emitter: emitter,
		upstreamURLs: map[string]string{
			"openai":    "https://api.openai.com",
			"anthropic": "https://api.anthropic.com",
		},
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/", s.handleProxy)
	return s
}

func (s *Server) SetUpstreamURL(provider, url string) {
	s.upstreamURLs[provider] = url
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Detect provider from path
	provider, upstreamPath := s.detectProvider(r.URL.Path)
	if provider == "" {
		http.Error(w, "unknown route", http.StatusBadGateway)
		return
	}

	// Extract and strip Corvade headers
	corvadeAgent := r.Header.Get("X-Corvade-Agent")
	corvadeSession := r.Header.Get("X-Corvade-Session")
	corvadeStep := r.Header.Get("X-Corvade-Step")
	r.Header.Del("X-Corvade-Agent")
	r.Header.Del("X-Corvade-Session")
	r.Header.Del("X-Corvade-Step")

	// Read request body
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Parse request to extract model
	var reqInfo *providers.RequestInfo
	switch provider {
	case "openai":
		reqInfo, _ = providers.OpenAIParseRequest(reqBody)
	case "anthropic":
		reqInfo, _ = providers.AnthropicParseRequest(reqBody)
	}

	model := ""
	if reqInfo != nil {
		model = reqInfo.Model
	}

	// Hash API key for grouping
	var apiKeyHash *string
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		authHeader = r.Header.Get("x-api-key")
	}
	if authHeader != "" {
		h := fmt.Sprintf("%x", sha256.Sum256([]byte(authHeader)))
		apiKeyHash = &h
	}

	// Forward request to upstream
	upstreamURL := s.upstreamURLs[provider] + upstreamPath
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	// Copy headers (except Corvade ones already stripped)
	for k, v := range r.Header {
		for _, vv := range v {
			upstreamReq.Header.Add(k, vv)
		}
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// Best effort — still try to forward what we got
		log.Printf("warning: failed to read upstream response: %v", err)
	}

	latencyMS := int(time.Since(start).Milliseconds())

	// Parse response for token counts
	var respInfo *providers.ResponseInfo
	switch provider {
	case "openai":
		respInfo, _ = providers.OpenAIParseResponse(respBody)
	case "anthropic":
		respInfo, _ = providers.AnthropicParseResponse(respBody)
	}

	// Write response to client
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Capture trace (best-effort)
	go s.captureTrace(provider, model, string(reqBody), string(respBody), resp.StatusCode,
		respInfo, latencyMS, apiKeyHash, corvadeAgent, corvadeSession, corvadeStep)
}

func (s *Server) detectProvider(path string) (provider, upstreamPath string) {
	if strings.HasPrefix(path, "/anthropic/") {
		return "anthropic", strings.TrimPrefix(path, "/anthropic")
	}
	if strings.HasPrefix(path, "/v1/") {
		return "openai", path
	}
	return "", ""
}

func (s *Server) captureTrace(provider, model, reqBody, respBody string, statusCode int,
	respInfo *providers.ResponseInfo, latencyMS int, apiKeyHash *string,
	agent, session, step string) {

	trace := &capture.Trace{
		ID:         capture.NewULID(),
		Provider:   provider,
		Model:      model,
		Request:    reqBody,
		Response:   &respBody,
		StatusCode: statusCode,
		LatencyMS:  &latencyMS,
		APIKeyHash: apiKeyHash,
		CreatedAt:  time.Now().UTC(),
	}

	if agent != "" {
		trace.Agent = &agent
	}
	if session != "" {
		trace.SessionID = &session
	}
	if step != "" {
		trace.Step = &step
	}

	if respInfo != nil {
		trace.TokensPrompt = &respInfo.PromptTokens
		trace.TokensCompletion = &respInfo.CompletionTokens
		trace.TokensCached = &respInfo.CachedTokens
		c := s.calc.Calculate(model, respInfo.PromptTokens, respInfo.CompletionTokens)
		trace.Cost = &c
	}

	if err := s.store.InsertTrace(trace); err != nil {
		log.Printf("warning: failed to capture trace: %v", err)
	}

	if s.emitter != nil {
		s.emitter.Emit("trace:new", trace)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/proxy/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/
git commit -m "feat: add HTTP reverse proxy with provider detection, capture, and header extraction"
```

---

## Task 9: WebSocket Event Hub

**Files:**
- Create: `internal/api/ws.go`
- Test: `internal/api/ws_test.go`

- [ ] **Step 1: Write WebSocket hub test**

```go
package api

import (
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch := make(chan []byte, 1)
	client := &Client{hub: hub, send: ch}
	hub.Register(client)

	time.Sleep(10 * time.Millisecond) // let goroutine process

	hub.Broadcast(Event{Type: "trace:new", Data: map[string]string{"id": "123"}})

	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast")
	}

	hub.Unregister(client)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/api/ -v
```

- [ ] **Step 3: Implement WebSocket hub**

```go
package api

import "encoding/json"

type Event struct {
	Type string      `json:"event"`
	Data interface{} `json:"data"`
}

type Client struct {
	hub  *Hub
	send chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) Broadcast(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.broadcast <- data
}

// Emit implements proxy.EventEmitter
func (h *Hub) Emit(eventType string, payload interface{}) {
	h.Broadcast(Event{Type: eventType, Data: payload})
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/ws.go internal/api/ws_test.go
git commit -m "feat: add WebSocket hub for real-time event broadcasting"
```

---

## Task 10: Dashboard REST API

**Files:**
- Create: `internal/api/server.go`, `internal/api/routes.go`, `internal/api/handlers/traces.go`, `internal/api/handlers/sessions.go`, `internal/api/handlers/topology.go`, `internal/api/handlers/export.go`
- Test: `internal/api/handlers/traces_test.go`

This is a larger task. The test covers the core trace listing endpoint; other endpoints follow the same pattern.

- [ ] **Step 1: Write traces handler test**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

func setupTestStore(t *testing.T) *capture.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestListTracesHandler(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Insert test data
	agent := "test-bot"
	store.InsertTrace(&capture.Trace{
		ID: "01HTEST000000000000000001", Agent: &agent, Provider: "openai",
		Model: "gpt-4o", Request: `{"model":"gpt-4o"}`, StatusCode: 200,
		CreatedAt: time.Now().UTC(),
	})

	h := NewTracesHandler(store)
	req := httptest.NewRequest("GET", "/api/traces?limit=10", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var traces []capture.Trace
	json.NewDecoder(w.Body).Decode(&traces)
	if len(traces) != 1 {
		t.Errorf("expected 1 trace, got %d", len(traces))
	}
}

func TestGetTraceHandler(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.InsertTrace(&capture.Trace{
		ID: "01HTEST000000000000000001", Provider: "openai", Model: "gpt-4o",
		Request: `{}`, StatusCode: 200, CreatedAt: time.Now().UTC(),
	})

	h := NewTracesHandler(store)
	req := httptest.NewRequest("GET", "/api/traces/01HTEST000000000000000001", nil)
	w := httptest.NewRecorder()

	h.Get(w, req, "01HTEST000000000000000001")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/api/handlers/ -v
```

- [ ] **Step 3: Implement handlers**

`internal/api/handlers/traces.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/corvade/corvade/internal/capture"
)

type TracesHandler struct {
	store *capture.Store
}

func NewTracesHandler(store *capture.Store) *TracesHandler {
	return &TracesHandler{store: store}
}

func (h *TracesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := capture.TraceFilter{
		Limit:  50,
		Offset: 0,
	}

	if v := q.Get("agent"); v != "" {
		filter.Agent = &v
	}
	if v := q.Get("model"); v != "" {
		filter.Model = &v
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	if v := q.Get("search"); v != "" {
		filter.Search = &v
	}

	traces, err := h.store.ListTraces(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traces)
}

func (h *TracesHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	trace, err := h.store.GetTrace(id)
	if err != nil {
		http.Error(w, "trace not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}
```

`internal/api/handlers/sessions.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/corvade/corvade/internal/capture"
)

type SessionsHandler struct {
	store *capture.Store
}

func NewSessionsHandler(store *capture.Store) *SessionsHandler {
	return &SessionsHandler{store: store}
}

func (h *SessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var agent *string
	if v := q.Get("agent"); v != "" {
		agent = &v
	}

	limit := 20
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	offset := 0
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}

	sessions, err := h.store.ListSessions(agent, nil, nil, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (h *SessionsHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	session, err := h.store.GetSession(id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}
```

`internal/api/handlers/topology.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/corvade/corvade/internal/capture"
)

type TopologyHandler struct {
	store *capture.Store
}

func NewTopologyHandler(store *capture.Store) *TopologyHandler {
	return &TopologyHandler{store: store}
}

type GraphResponse struct {
	Nodes []capture.GraphNode `json:"nodes"`
	Edges []capture.GraphEdge `json:"edges"`
}

func (h *TopologyHandler) GetGraph(w http.ResponseWriter, r *http.Request, sessionID string) {
	nodes, edges, err := h.store.GetSessionGraph(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GraphResponse{Nodes: nodes, Edges: edges})
}
```

`internal/api/handlers/export.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/corvade/corvade/internal/capture"
)

type ExportHandler struct {
	store *capture.Store
}

func NewExportHandler(store *capture.Store) *ExportHandler {
	return &ExportHandler{store: store}
}

type StatsResponse struct {
	TotalTraces   int                `json:"total_traces"`
	TotalSessions int                `json:"total_sessions"`
	TotalCost     float64            `json:"total_cost"`
	TotalTokens   int                `json:"total_tokens"`
	ByModel       map[string]int     `json:"by_model"`
	ByAgent       map[string]int     `json:"by_agent"`
}

func (h *ExportHandler) Stats(w http.ResponseWriter, r *http.Request) {
	traces, _ := h.store.ListTraces(capture.TraceFilter{Limit: 10000})
	sessions, _ := h.store.ListSessions(nil, nil, nil, 10000, 0)

	stats := StatsResponse{
		TotalTraces:   len(traces),
		TotalSessions: len(sessions),
		ByModel:       make(map[string]int),
		ByAgent:       make(map[string]int),
	}

	for _, t := range traces {
		stats.ByModel[t.Model]++
		if t.Agent != nil {
			stats.ByAgent[*t.Agent]++
		}
		if t.Cost != nil {
			stats.TotalCost += *t.Cost
		}
		if t.TokensPrompt != nil {
			stats.TotalTokens += *t.TokensPrompt
		}
		if t.TokensCompletion != nil {
			stats.TotalTokens += *t.TokensCompletion
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
```

`internal/api/routes.go`:

```go
package api

import (
	"net/http"
	"strings"

	"github.com/corvade/corvade/internal/api/handlers"
	"github.com/corvade/corvade/internal/capture"
)

func RegisterRoutes(mux *http.ServeMux, store *capture.Store) {
	traces := handlers.NewTracesHandler(store)
	sessions := handlers.NewSessionsHandler(store)
	topology := handlers.NewTopologyHandler(store)
	export := handlers.NewExportHandler(store)

	mux.HandleFunc("/api/traces", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/traces" {
			traces.List(w, r)
			return
		}
		// /api/traces/{id}
		id := strings.TrimPrefix(r.URL.Path, "/api/traces/")
		traces.Get(w, r, id)
	})

	mux.HandleFunc("/api/traces/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/traces/")
		traces.Get(w, r, id)
	})

	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions.List(w, r)
	})

	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		parts := strings.Split(path, "/")

		if len(parts) == 1 {
			sessions.Get(w, r, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "graph" {
			topology.GetGraph(w, r, parts[0])
			return
		}
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/stats", export.Stats)
}
```

`internal/api/server.go`:

```go
package api

import (
	"fmt"
	"net/http"

	"github.com/corvade/corvade/internal/capture"
)

type APIServer struct {
	store *capture.Store
	hub   *Hub
	mux   *http.ServeMux
	port  int
}

func NewAPIServer(store *capture.Store, hub *Hub, port int) *APIServer {
	mux := http.NewServeMux()
	RegisterRoutes(mux, store)

	return &APIServer{
		store: store,
		hub:   hub,
		mux:   mux,
		port:  port,
	}
}

func (s *APIServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, s.mux)
}

func (s *APIServer) Handler() http.Handler {
	return s.mux
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/
git commit -m "feat: add REST API server with traces, sessions, topology, and stats endpoints"
```

---

## Task 11: CLI — Start Command

**Files:**
- Create: `internal/cli/start.go`
- Modify: `cmd/corvade/main.go`

- [ ] **Step 1: Implement start command**

```go
package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/corvade/corvade/internal/api"
	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/config"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy"
	"github.com/spf13/cobra"
)

func NewStartCmd(version string) *cobra.Command {
	var headless bool
	var demo bool
	var port int
	var dashboardPort int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Corvade proxy and dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfgPath := config.DefaultPath()
			cfg, created, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if created {
				fmt.Printf("  First run — created config at %s\n\n", cfgPath)
			}

			// Override from flags
			if port != 0 {
				cfg.Port = port
			}
			if dashboardPort != 0 {
				cfg.DashboardPort = dashboardPort
			}

			// Open store
			dbDir := filepath.Join(os.Getenv("HOME"), ".corvade")
			os.MkdirAll(dbDir, 0755)
			dbPath := filepath.Join(dbDir, "corvade.db")

			store, err := capture.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer store.Close()

			// Build cost overrides from config
			var costOverrides map[string]cost.ModelPricing
			if len(cfg.CostOverrides) > 0 {
				costOverrides = make(map[string]cost.ModelPricing)
				for model, override := range cfg.CostOverrides {
					costOverrides[model] = cost.ModelPricing{
						InputPer1K:  override.InputPer1K,
						OutputPer1K: override.OutputPer1K,
					}
				}
			}
			calc := cost.NewCalculator(costOverrides)

			// Start WebSocket hub
			hub := api.NewHub()
			go hub.Run()

			// Start proxy
			proxySrv := proxy.NewServer(store, calc, hub)
			go func() {
				addr := fmt.Sprintf(":%d", cfg.Port)
				if err := http.ListenAndServe(addr, proxySrv); err != nil {
					log.Fatalf("proxy error: %v", err)
				}
			}()

			// Start API server
			if !headless {
				apiSrv := api.NewAPIServer(store, hub, cfg.DashboardPort)
				go func() {
					if err := apiSrv.Start(); err != nil {
						log.Fatalf("API server error: %v", err)
					}
				}()
			}

			// Print banner
			dbInfo, _ := os.Stat(dbPath)
			dbSize := "new"
			if dbInfo != nil {
				dbSize = fmt.Sprintf("%.1f MB", float64(dbInfo.Size())/1024/1024)
			}

			fmt.Printf("\n")
			fmt.Printf("  ▗▖  Corvade v%s\n", version)
			fmt.Printf("  ▝▘  The AI agent control plane\n")
			fmt.Printf("\n")
			fmt.Printf("  Proxy:      http://localhost:%d\n", cfg.Port)
			if !headless {
				fmt.Printf("  Dashboard:  http://localhost:%d\n", cfg.DashboardPort)
			}
			fmt.Printf("  Storage:    %s (%s)\n", dbPath, dbSize)
			fmt.Printf("\n")
			fmt.Printf("  Point your agents here:\n")
			fmt.Printf("    OPENAI_BASE_URL=http://localhost:%d/v1\n", cfg.Port)
			fmt.Printf("    ANTHROPIC_BASE_URL=http://localhost:%d/anthropic\n", cfg.Port)
			fmt.Printf("\n")
			fmt.Printf("  Watching for traffic...\n")

			// Wait for interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			fmt.Println("\nShutting down...")
			return nil
		},
	}

	cmd.Flags().BoolVar(&headless, "headless", false, "Proxy only, no dashboard")
	cmd.Flags().BoolVar(&demo, "demo", false, "Start with demo fixture data")
	cmd.Flags().IntVar(&port, "port", 0, "Override proxy port")
	cmd.Flags().IntVar(&dashboardPort, "dashboard-port", 0, "Override dashboard port")

	return cmd
}
```

- [ ] **Step 2: Wire start command into main.go**

Update `cmd/corvade/main.go` to add:

```go
import "github.com/corvade/corvade/internal/cli"

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cli.NewStartCmd(version))
}
```

- [ ] **Step 3: Build and verify**

```bash
make build
./bin/corvade start --help
```

Expected: shows help with `--headless`, `--demo`, `--port`, `--dashboard-port` flags

- [ ] **Step 4: Commit**

```bash
git add internal/cli/start.go cmd/corvade/main.go
git commit -m "feat: add corvade start command with proxy and API server startup"
```

---

## Task 12: CLI — Tail, Doctor, Sessions, Inspect

**Files:**
- Create: `internal/cli/tail.go`, `internal/cli/doctor.go`, `internal/cli/sessions.go`, `internal/cli/inspect.go`
- Modify: `cmd/corvade/main.go`

- [ ] **Step 1: Implement tail command**

`internal/cli/tail.go` — connects to WebSocket and prints live trace feed to terminal:

```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

func NewTailCmd() *cobra.Command {
	var agent string
	var model string
	var port int

	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Live stream of intercepted calls",
		RunE: func(cmd *cobra.Command, args []string) error {
			if port == 0 {
				port = 4401
			}

			url := fmt.Sprintf("ws://localhost:%d/ws", port)
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				return fmt.Errorf("connecting to dashboard: %w\n  Is corvade running? Try: corvade start", err)
			}
			defer conn.Close()

			fmt.Fprintf(os.Stderr, "Connected to Corvade. Watching for traffic...\n\n")

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return nil
				}

				var event struct {
					Type string          `json:"event"`
					Data json.RawMessage `json:"data"`
				}
				json.Unmarshal(msg, &event)

				if event.Type != "trace:new" {
					continue
				}

				var trace struct {
					Model      string   `json:"model"`
					Agent      *string  `json:"agent"`
					Cost       *float64 `json:"cost"`
					LatencyMS  *int     `json:"latency_ms"`
					StatusCode int      `json:"status_code"`
					Step       *string  `json:"step"`
					CreatedAt  string   `json:"created_at"`
				}
				json.Unmarshal(event.Data, &trace)

				// Apply filters
				if agent != "" && (trace.Agent == nil || *trace.Agent != agent) {
					continue
				}
				if model != "" && !strings.Contains(trace.Model, model) {
					continue
				}

				// Format output
				status := "✓"
				if trace.StatusCode >= 400 {
					status = "✗"
				}
				costStr := "    -"
				if trace.Cost != nil {
					costStr = fmt.Sprintf("$%.2f", *trace.Cost)
				}
				latStr := "   -"
				if trace.LatencyMS != nil {
					latStr = fmt.Sprintf("%.1fs", float64(*trace.LatencyMS)/1000)
				}
				stepStr := ""
				if trace.Step != nil {
					stepStr = *trace.Step
				}

				fmt.Printf("%-13s │ %5s │ %4s │ %s %s\n",
					trace.Model, costStr, latStr, status, stepStr)
			}
		},
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Filter by agent name")
	cmd.Flags().StringVar(&model, "model", "", "Filter by model (substring match)")
	cmd.Flags().IntVar(&port, "port", 0, "Dashboard port (default 4401)")

	return cmd
}
```

- [ ] **Step 2: Implement doctor command**

`internal/cli/doctor.go`:

```go
package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/corvade/corvade/internal/config"
	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _ := config.Load(config.DefaultPath())

			fmt.Println()

			// Check proxy port
			checkPort(cfg.Port, "Proxy port")

			// Check dashboard port
			checkPort(cfg.DashboardPort, "Dashboard port")

			// Check SQLite
			dbPath := filepath.Join(os.Getenv("HOME"), ".corvade", "corvade.db")
			if _, err := os.Stat(dbPath); err == nil {
				f, err := os.OpenFile(dbPath, os.O_WRONLY, 0)
				if err != nil {
					fmt.Printf("  ✗ SQLite not writable at %s\n", dbPath)
				} else {
					f.Close()
					fmt.Printf("  ✓ SQLite writable at %s\n", dbPath)
				}
			} else {
				fmt.Printf("  - No database yet at %s (will be created on first start)\n", dbPath)
			}

			// Check disk space
			var stat syscall.Statfs_t
			home, _ := os.UserHomeDir()
			if err := syscall.Statfs(home, &stat); err == nil {
				availGB := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
				if availGB < 1 {
					fmt.Printf("  ✗ Low disk space: %.1f GB available\n", availGB)
				} else {
					fmt.Printf("  ✓ %.1f GB disk space available\n", availGB)
				}
			}

			fmt.Println()
			return nil
		},
	}
}

func checkPort(port int, name string) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("  ✗ %s %d is in use\n", name, port)
	} else {
		ln.Close()
		fmt.Printf("  ✓ %s %d available\n", name, port)
	}
}
```

- [ ] **Step 3: Implement sessions command**

`internal/cli/sessions.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/corvade/corvade/internal/capture"
	"github.com/spf13/cobra"
)

func NewSessionsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List recent sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(os.Getenv("HOME"), ".corvade", "corvade.db")
			store, err := capture.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer store.Close()

			sessions, err := store.ListSessions(nil, nil, nil, limit, 0)
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tAGENT\tTRACES\tCOST\tSTATUS\tSTARTED")
			for _, s := range sessions {
				agent := "-"
				if s.Agent != nil {
					agent = *s.Agent
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t$%.4f\t%s\t%s\n",
					s.ID[:12], agent, s.TraceCount, s.TotalCost, s.Status,
					s.StartTime.Format("15:04:05"))
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of sessions to show")
	return cmd
}
```

- [ ] **Step 4: Implement inspect command**

`internal/cli/inspect.go`:

```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/corvade/corvade/internal/capture"
	"github.com/spf13/cobra"
)

func NewInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <id>",
		Short: "Show full detail of a trace or session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			dbPath := filepath.Join(os.Getenv("HOME"), ".corvade", "corvade.db")
			store, err := capture.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer store.Close()

			// Try trace first
			trace, err := store.GetTrace(id)
			if err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(trace)
			}

			// Try session
			session, err := store.GetSession(id)
			if err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(session)
			}

			return fmt.Errorf("no trace or session found with ID: %s", id)
		},
	}
}
```

- [ ] **Step 5: Wire all commands into main.go**

Update `cmd/corvade/main.go` init():

```go
func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cli.NewStartCmd(version))
	rootCmd.AddCommand(cli.NewTailCmd())
	rootCmd.AddCommand(cli.NewDoctorCmd())
	rootCmd.AddCommand(cli.NewSessionsCmd())
	rootCmd.AddCommand(cli.NewInspectCmd())
}
```

- [ ] **Step 6: Install gorilla/websocket and build**

```bash
go get github.com/gorilla/websocket
make build
./bin/corvade --help
```

Expected: all commands listed

- [ ] **Step 7: Commit**

```bash
git add internal/cli/ cmd/corvade/main.go go.mod go.sum
git commit -m "feat: add tail, doctor, sessions, and inspect CLI commands"
```

---

## Task 13: Topology Inference Engine

**Files:**
- Create: `internal/topology/inference.go`, `internal/topology/fingerprint.go`, `internal/topology/toolcall.go`, `internal/topology/graph.go`
- Test: `internal/topology/inference_test.go`
- Create: `testdata/tool-call-chain.json`, `testdata/multi-turn-conversation.json`

- [ ] **Step 1: Create test fixtures**

`testdata/tool-call-chain.json` — a sequence of traces representing: LLM call → tool call → tool result → follow-up LLM call:

```json
[
  {
    "id": "01TRACE_001",
    "provider": "openai",
    "model": "gpt-4o",
    "request": "{\"model\":\"gpt-4o\",\"messages\":[{\"role\":\"user\",\"content\":\"search for AI agents\"}]}",
    "response": "{\"choices\":[{\"message\":{\"tool_calls\":[{\"id\":\"call_abc\",\"function\":{\"name\":\"web_search\",\"arguments\":\"{\\\"q\\\":\\\"AI agents\\\"}\"}}]}}],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":15}}",
    "status_code": 200,
    "created_at": "2026-03-18T10:00:01Z"
  },
  {
    "id": "01TRACE_002",
    "provider": "openai",
    "model": "gpt-4o",
    "request": "{\"model\":\"gpt-4o\",\"messages\":[{\"role\":\"user\",\"content\":\"search for AI agents\"},{\"role\":\"assistant\",\"tool_calls\":[{\"id\":\"call_abc\"}]},{\"role\":\"tool\",\"tool_call_id\":\"call_abc\",\"content\":\"found 5 results\"}]}",
    "response": "{\"choices\":[{\"message\":{\"content\":\"Here are the results...\"}}],\"usage\":{\"prompt_tokens\":50,\"completion_tokens\":30}}",
    "status_code": 200,
    "created_at": "2026-03-18T10:00:03Z"
  }
]
```

- [ ] **Step 2: Write inference test**

```go
package topology

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

type testTrace struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Request   string `json:"request"`
	Response  string `json:"response"`
	StatusCode int   `json:"status_code"`
	CreatedAt string `json:"created_at"`
}

func loadFixture(t *testing.T, path string) []capture.Trace {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var raw []testTrace
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	var traces []capture.Trace
	for _, r := range raw {
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
		traces = append(traces, capture.Trace{
			ID:         r.ID,
			Provider:   r.Provider,
			Model:      r.Model,
			Request:    r.Request,
			Response:   &r.Response,
			StatusCode: r.StatusCode,
			CreatedAt:  createdAt,
		})
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

	// Should detect a "triggered" edge from first call to second
	if len(edges) < 1 {
		t.Fatalf("expected at least 1 edge, got %d", len(edges))
	}

	foundTriggered := false
	for _, e := range edges {
		if e.Type == "triggered" {
			foundTriggered = true
		}
	}
	if !foundTriggered {
		t.Error("expected a 'triggered' edge from tool call chain")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/topology/ -v
```

- [ ] **Step 4: Implement inference engine**

`internal/topology/fingerprint.go`:

```go
package topology

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// FingerprintMessages hashes the message array from a request body.
// Returns empty string if messages can't be extracted.
func FingerprintMessages(requestBody string) string {
	var req struct {
		Messages json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(requestBody), &req); err != nil || req.Messages == nil {
		return ""
	}
	hash := sha256.Sum256(req.Messages)
	return fmt.Sprintf("%x", hash)
}

// ContainsAssistantResponse checks if the messages in requestB contain
// content from responseA, indicating B is a follow-up to A.
func ContainsAssistantResponse(requestB string, responseA string) bool {
	if responseA == "" {
		return false
	}

	// Extract assistant content from responseA
	var respA struct {
		Choices []struct {
			Message struct {
				Content   string          `json:"content"`
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		// Anthropic format
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	json.Unmarshal([]byte(responseA), &respA)

	// Get some identifying content from the response
	var marker string
	if len(respA.Choices) > 0 {
		if respA.Choices[0].Message.Content != "" {
			marker = respA.Choices[0].Message.Content
		} else if respA.Choices[0].Message.ToolCalls != nil {
			marker = string(respA.Choices[0].Message.ToolCalls)
		}
	}
	if len(respA.Content) > 0 {
		marker = respA.Content[0].Text
	}

	if marker == "" {
		return false
	}

	// Check if requestB's messages contain this marker
	return len(marker) > 5 && containsSubstring(requestB, marker[:min(len(marker), 50)])
}

func containsSubstring(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) > 0 && indexOf(haystack, needle) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

`internal/topology/toolcall.go`:

```go
package topology

import "encoding/json"

// ExtractToolCallIDs extracts tool_call IDs from a response body.
func ExtractToolCallIDs(responseBody string) []string {
	// OpenAI format
	var openai struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID string `json:"id"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(responseBody), &openai); err == nil {
		if len(openai.Choices) > 0 {
			var ids []string
			for _, tc := range openai.Choices[0].Message.ToolCalls {
				if tc.ID != "" {
					ids = append(ids, tc.ID)
				}
			}
			if len(ids) > 0 {
				return ids
			}
		}
	}

	// Anthropic format
	var anthropic struct {
		Content []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(responseBody), &anthropic); err == nil {
		var ids []string
		for _, c := range anthropic.Content {
			if c.Type == "tool_use" && c.ID != "" {
				ids = append(ids, c.ID)
			}
		}
		return ids
	}

	return nil
}

// ExtractReferencedToolCallIDs extracts tool_call_ids referenced in a request's messages.
func ExtractReferencedToolCallIDs(requestBody string) []string {
	var req struct {
		Messages []struct {
			Role       string `json:"role"`
			ToolCallID string `json:"tool_call_id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(requestBody), &req); err != nil {
		return nil
	}

	var ids []string
	for _, m := range req.Messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			ids = append(ids, m.ToolCallID)
		}
	}
	return ids
}
```

`internal/topology/graph.go`:

```go
package topology

import (
	"time"

	"github.com/corvade/corvade/internal/capture"
)

func buildNode(trace capture.Trace, nodeType string, confidence float64) capture.GraphNode {
	snapshot := trace.Request
	if len(snapshot) > 500 {
		snapshot = snapshot[:500]
	}

	return capture.GraphNode{
		ID:              capture.NewULID(),
		TraceID:         trace.ID,
		SessionID:       stringOrDefault(trace.SessionID, "default"),
		Type:            nodeType,
		Agent:           trace.Agent,
		Step:            trace.Step,
		Model:           &trace.Model,
		Tokens:          sumTokens(trace.TokensPrompt, trace.TokensCompletion),
		Cost:            trace.Cost,
		LatencyMS:       trace.LatencyMS,
		ContextSnapshot: &snapshot,
		Confidence:      confidence,
		CreatedAt:       trace.CreatedAt,
	}
}

func buildEdge(sessionID, fromNode, toNode, edgeType string, confidence float64) capture.GraphEdge {
	return capture.GraphEdge{
		ID:         capture.NewULID(),
		SessionID:  sessionID,
		FromNode:   fromNode,
		ToNode:     toNode,
		Type:       edgeType,
		Confidence: confidence,
		CreatedAt:  time.Now().UTC(),
	}
}

func stringOrDefault(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

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
```

`internal/topology/inference.go`:

```go
package topology

import (
	"github.com/corvade/corvade/internal/capture"
)

type Engine struct {
	timingGapMS int
}

func NewEngine(timingGapMS int) *Engine {
	return &Engine{timingGapMS: timingGapMS}
}

func (e *Engine) Infer(traces []capture.Trace) ([]capture.GraphNode, []capture.GraphEdge) {
	if len(traces) == 0 {
		return nil, nil
	}

	var nodes []capture.GraphNode
	var edges []capture.GraphEdge

	// Build a node for each trace
	traceToNode := make(map[string]string) // trace ID → node ID
	for _, t := range traces {
		node := buildNode(t, "llm_call", 1.0)
		traceToNode[t.ID] = node.ID
		nodes = append(nodes, node)
	}

	// Build tool call ID index: response tool_call_id → trace ID
	toolCallIndex := make(map[string]string) // tool_call_id → trace ID that produced it
	for _, t := range traces {
		if t.Response == nil {
			continue
		}
		ids := ExtractToolCallIDs(*t.Response)
		for _, id := range ids {
			toolCallIndex[id] = t.ID
		}
	}

	// Strategy 1: Tool call ID chain resolution
	for _, t := range traces {
		refIDs := ExtractReferencedToolCallIDs(t.Request)
		for _, refID := range refIDs {
			if parentTraceID, ok := toolCallIndex[refID]; ok {
				if parentNodeID, ok := traceToNode[parentTraceID]; ok {
					if childNodeID, ok := traceToNode[t.ID]; ok {
						sessionID := stringOrDefault(t.SessionID, "default")
						edge := buildEdge(sessionID, parentNodeID, childNodeID, "triggered", 0.9)
						edges = append(edges, edge)
					}
				}
			}
		}
	}

	// Strategy 2: Conversation fingerprinting
	for i := 1; i < len(traces); i++ {
		prev := traces[i-1]
		curr := traces[i]

		// Skip if already connected by tool call chain
		if hasEdgeBetween(edges, traceToNode[prev.ID], traceToNode[curr.ID]) {
			continue
		}

		if prev.Response != nil && ContainsAssistantResponse(curr.Request, *prev.Response) {
			sessionID := stringOrDefault(curr.SessionID, "default")
			edge := buildEdge(sessionID, traceToNode[prev.ID], traceToNode[curr.ID], "triggered", 0.7)
			edges = append(edges, edge)
		}
	}

	// Strategy 3: Timing gap (for sequential traces without other signals)
	for i := 1; i < len(traces); i++ {
		prev := traces[i-1]
		curr := traces[i]

		if hasEdgeBetween(edges, traceToNode[prev.ID], traceToNode[curr.ID]) {
			continue
		}

		gapMS := curr.CreatedAt.Sub(prev.CreatedAt).Milliseconds()
		if gapMS < int64(e.timingGapMS) && gapMS >= 0 {
			sessionID := stringOrDefault(curr.SessionID, "default")
			confidence := 0.5
			if gapMS < 500 {
				confidence = 0.6
			}
			edge := buildEdge(sessionID, traceToNode[prev.ID], traceToNode[curr.ID], "triggered", confidence)
			edges = append(edges, edge)
		}
	}

	return nodes, edges
}

func hasEdgeBetween(edges []capture.GraphEdge, fromNode, toNode string) bool {
	for _, e := range edges {
		if e.FromNode == fromNode && e.ToNode == toNode {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/topology/ -v
```

Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/topology/ testdata/
git commit -m "feat: add topology inference engine with fingerprinting, tool call tracking, and timing gaps"
```

---

## Task 14: Next.js Dashboard Scaffolding

**Files:**
- Create: `dashboard/` (Next.js project with Tailwind, React Flow)

- [ ] **Step 1: Initialize Next.js project**

```bash
cd ~/projects-wsl/corvade
npx create-next-app@latest dashboard --typescript --tailwind --app --no-src-dir --no-eslint --no-import-alias
```

Wait — the spec says `dashboard/src/`. Let's use the src directory:

```bash
npx create-next-app@latest dashboard --typescript --tailwind --app --src-dir --no-eslint --no-import-alias
```

- [ ] **Step 2: Install dependencies**

```bash
cd dashboard
npm install @xyflow/react dagre
npm install -D @types/dagre
```

- [ ] **Step 3: Create API client**

`dashboard/src/lib/api.ts`:

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

export async function fetchTraces(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/traces`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  }
  const res = await fetch(url.toString());
  return res.json();
}

export async function fetchTrace(id: string) {
  const res = await fetch(`${API_BASE}/api/traces/${id}`);
  return res.json();
}

export async function fetchSessions(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/sessions`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  }
  const res = await fetch(url.toString());
  return res.json();
}

export async function fetchSession(id: string) {
  const res = await fetch(`${API_BASE}/api/sessions/${id}`);
  return res.json();
}

export async function fetchSessionGraph(id: string) {
  const res = await fetch(`${API_BASE}/api/sessions/${id}/graph`);
  return res.json();
}

export async function fetchStats(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/stats`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  }
  const res = await fetch(url.toString());
  return res.json();
}
```

- [ ] **Step 4: Create WebSocket client**

`dashboard/src/lib/ws.ts`:

```typescript
'use client';

import { createContext, useContext, useEffect, useRef, useState, ReactNode } from 'react';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:4401/ws';

interface WSEvent {
  event: string;
  data: any;
}

type EventHandler = (data: any) => void;

class CorvadeWS {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Set<EventHandler>> = new Map();
  private reconnectTimer: NodeJS.Timeout | null = null;

  connect() {
    this.ws = new WebSocket(WS_URL);
    this.ws.onmessage = (e) => {
      const event: WSEvent = JSON.parse(e.data);
      const handlers = this.handlers.get(event.event);
      if (handlers) {
        handlers.forEach(h => h(event.data));
      }
    };
    this.ws.onclose = () => {
      this.reconnectTimer = setTimeout(() => this.connect(), 2000);
    };
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
  }

  on(event: string, handler: EventHandler) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
  }

  off(event: string, handler: EventHandler) {
    this.handlers.get(event)?.delete(handler);
  }
}

export { CorvadeWS };
```

- [ ] **Step 5: Verify dashboard builds**

```bash
cd ~/projects-wsl/corvade/dashboard
npm run build
```

Expected: build succeeds

- [ ] **Step 6: Commit**

```bash
cd ~/projects-wsl/corvade
git add dashboard/
git commit -m "feat: scaffold Next.js dashboard with API client and WebSocket"
```

---

## Task 15: Timeline Component

**Files:**
- Create: `dashboard/src/components/Timeline.tsx`
- Modify: `dashboard/src/app/page.tsx`

- [ ] **Step 1: Build Timeline component**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { fetchTraces } from '@/lib/api';

interface Trace {
  id: string;
  model: string;
  agent: string | null;
  step: string | null;
  tokens_prompt: number | null;
  tokens_completion: number | null;
  cost: number | null;
  latency_ms: number | null;
  status_code: number;
  created_at: string;
}

export default function Timeline() {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [filter, setFilter] = useState({ agent: '', model: '', search: '' });

  useEffect(() => {
    const params: Record<string, string> = { limit: '50' };
    if (filter.agent) params.agent = filter.agent;
    if (filter.model) params.model = filter.model;
    if (filter.search) params.search = filter.search;

    fetchTraces(params).then(setTraces).catch(console.error);
  }, [filter]);

  const statusColor = (code: number) => {
    if (code >= 400) return 'text-red-400';
    if (code === 429) return 'text-yellow-400';
    return 'text-green-400';
  };

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex gap-3">
        <input
          type="text"
          placeholder="Filter by agent..."
          className="bg-zinc-800 border border-zinc-700 rounded px-3 py-1.5 text-sm text-zinc-200"
          value={filter.agent}
          onChange={(e) => setFilter(f => ({ ...f, agent: e.target.value }))}
        />
        <input
          type="text"
          placeholder="Filter by model..."
          className="bg-zinc-800 border border-zinc-700 rounded px-3 py-1.5 text-sm text-zinc-200"
          value={filter.model}
          onChange={(e) => setFilter(f => ({ ...f, model: e.target.value }))}
        />
        <input
          type="text"
          placeholder="Search prompts..."
          className="bg-zinc-800 border border-zinc-700 rounded px-3 py-1.5 text-sm text-zinc-200 flex-1"
          value={filter.search}
          onChange={(e) => setFilter(f => ({ ...f, search: e.target.value }))}
        />
      </div>

      {/* Trace list */}
      <div className="border border-zinc-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-zinc-900 text-zinc-400">
            <tr>
              <th className="text-left px-4 py-2">Time</th>
              <th className="text-left px-4 py-2">Model</th>
              <th className="text-left px-4 py-2">Agent</th>
              <th className="text-right px-4 py-2">Tokens</th>
              <th className="text-right px-4 py-2">Cost</th>
              <th className="text-right px-4 py-2">Latency</th>
              <th className="text-center px-4 py-2">Status</th>
            </tr>
          </thead>
          <tbody>
            {traces.map((trace) => (
              <tr key={trace.id} className="border-t border-zinc-800 hover:bg-zinc-900/50 cursor-pointer">
                <td className="px-4 py-2 text-zinc-400 font-mono text-xs">
                  {new Date(trace.created_at).toLocaleTimeString()}
                </td>
                <td className="px-4 py-2 text-zinc-200">{trace.model}</td>
                <td className="px-4 py-2 text-zinc-400">{trace.agent || '-'}</td>
                <td className="px-4 py-2 text-right text-zinc-400 font-mono">
                  {((trace.tokens_prompt || 0) + (trace.tokens_completion || 0)).toLocaleString()}
                </td>
                <td className="px-4 py-2 text-right text-zinc-400 font-mono">
                  {trace.cost != null ? `$${trace.cost.toFixed(4)}` : '-'}
                </td>
                <td className="px-4 py-2 text-right text-zinc-400 font-mono">
                  {trace.latency_ms != null ? `${(trace.latency_ms / 1000).toFixed(1)}s` : '-'}
                </td>
                <td className={`px-4 py-2 text-center font-mono ${statusColor(trace.status_code)}`}>
                  {trace.status_code}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {traces.length === 0 && (
          <div className="text-center py-12 text-zinc-500">
            No traces captured yet. Point your agents at the proxy to get started.
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Update home page**

`dashboard/src/app/page.tsx`:

```tsx
import Timeline from '@/components/Timeline';

export default function Home() {
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <h1 className="text-xl font-semibold">Corvade</h1>
          <span className="text-zinc-500 text-sm">AI Agent Control Plane</span>
        </div>
        <Timeline />
      </div>
    </main>
  );
}
```

- [ ] **Step 3: Verify it builds**

```bash
cd ~/projects-wsl/corvade/dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
cd ~/projects-wsl/corvade
git add dashboard/
git commit -m "feat: add Timeline component with filtering and trace list"
```

---

## Task 16: Topology Graph Component

**Files:**
- Create: `dashboard/src/components/TopologyGraph.tsx`
- Create: `dashboard/src/app/session/[id]/page.tsx`

- [ ] **Step 1: Build TopologyGraph component**

```tsx
'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  Node,
  Edge,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import { fetchSessionGraph } from '@/lib/api';

interface GraphNode {
  id: string;
  type: string;
  agent: string | null;
  step: string | null;
  model: string | null;
  tokens: number | null;
  cost: number | null;
  latency_ms: number | null;
  confidence: number;
}

interface GraphEdge {
  id: string;
  from_node: string;
  to_node: string;
  type: string;
  confidence: number;
}

const NODE_WIDTH = 220;
const NODE_HEIGHT = 80;

const nodeColors: Record<string, string> = {
  llm_call: '#3b82f6',
  tool_call: '#f59e0b',
  tool_result: '#10b981',
  decision_point: '#ef4444',
};

function layoutGraph(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'TB', nodesep: 50, ranksep: 80 });

  nodes.forEach((node) => g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT }));
  edges.forEach((edge) => g.setEdge(edge.source, edge.target));

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
    };
  });
}

export default function TopologyGraph({ sessionId }: { sessionId: string }) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    fetchSessionGraph(sessionId).then((data: { nodes: GraphNode[]; edges: GraphEdge[] }) => {
      const flowNodes: Node[] = (data.nodes || []).map((n) => ({
        id: n.id,
        type: 'default',
        data: {
          label: (
            <div className="text-xs">
              <div className="font-bold">{n.model || 'unknown'}</div>
              <div className="text-zinc-400">{n.step || n.type}</div>
              {n.cost != null && <div className="text-zinc-500">${n.cost.toFixed(4)}</div>}
            </div>
          ),
        },
        position: { x: 0, y: 0 },
        sourcePosition: Position.Bottom,
        targetPosition: Position.Top,
        style: {
          background: '#18181b',
          border: `2px solid ${nodeColors[n.type] || '#71717a'}`,
          borderRadius: n.type === 'decision_point' ? '0' : '8px',
          borderStyle: n.confidence < 0.8 ? 'dashed' : 'solid',
          padding: '8px 12px',
          color: '#e4e4e7',
          width: NODE_WIDTH,
        },
      }));

      const flowEdges: Edge[] = (data.edges || []).map((e) => ({
        id: e.id,
        source: e.from_node,
        target: e.to_node,
        label: e.type,
        style: {
          stroke: e.confidence < 0.8 ? '#52525b' : '#a1a1aa',
          strokeDasharray: e.confidence < 0.8 ? '5,5' : undefined,
        },
        labelStyle: { fill: '#71717a', fontSize: 10 },
      }));

      const laidOut = layoutGraph(flowNodes, flowEdges);
      setNodes(laidOut);
      setEdges(flowEdges);
    });
  }, [sessionId]);

  return (
    <div className="h-[600px] bg-zinc-950 rounded-lg border border-zinc-800">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#27272a" gap={20} />
        <Controls />
      </ReactFlow>
    </div>
  );
}
```

- [ ] **Step 2: Create session detail page**

`dashboard/src/app/session/[id]/page.tsx`:

```tsx
import TopologyGraph from '@/components/TopologyGraph';

export default async function SessionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <a href="/" className="text-zinc-500 hover:text-zinc-300">← Back</a>
          <h1 className="text-xl font-semibold">Session {id.slice(0, 12)}</h1>
        </div>
        <TopologyGraph sessionId={id} />
      </div>
    </main>
  );
}
```

- [ ] **Step 3: Verify it builds**

```bash
cd ~/projects-wsl/corvade/dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
cd ~/projects-wsl/corvade
git add dashboard/
git commit -m "feat: add TopologyGraph component with Dagre layout and session detail page"
```

---

## Task 17: Narrative View Component

**Files:**
- Create: `dashboard/src/lib/narrative.ts`, `dashboard/src/components/NarrativeView.tsx`

- [ ] **Step 1: Build narrative template engine**

`dashboard/src/lib/narrative.ts`:

```typescript
interface NarrativeNode {
  id: string;
  type: string;
  agent: string | null;
  step: string | null;
  model: string | null;
  tokens: number | null;
  cost: number | null;
  latency_ms: number | null;
}

export interface NarrativeSegment {
  nodeId: string;
  text: string;
  nodeType: string;
}

export function generateNarrative(nodes: NarrativeNode[]): NarrativeSegment[] {
  return nodes.map((node) => {
    const agent = node.agent || 'The agent';
    const model = node.model || 'the LLM';
    const step = node.step || 'process the request';
    const costStr = node.cost != null ? ` (cost: $${node.cost.toFixed(4)})` : '';
    const latencyStr = node.latency_ms != null ? ` in ${(node.latency_ms / 1000).toFixed(1)}s` : '';

    let text: string;
    switch (node.type) {
      case 'llm_call':
        text = `${agent} asked ${model} to ${step}${latencyStr}${costStr}.`;
        break;
      case 'tool_call':
        text = `It called ${step || 'a tool'}${latencyStr}.`;
        break;
      case 'tool_result':
        text = `It received the result from ${step || 'the tool'}.`;
        break;
      case 'decision_point':
        text = `It decided to ${step || 'take an action'}.`;
        break;
      default:
        text = `${agent} performed ${node.type}${latencyStr}${costStr}.`;
    }

    return { nodeId: node.id, text, nodeType: node.type };
  });
}
```

- [ ] **Step 2: Build NarrativeView component**

`dashboard/src/components/NarrativeView.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { generateNarrative, NarrativeSegment } from '@/lib/narrative';

interface GraphNode {
  id: string;
  type: string;
  agent: string | null;
  step: string | null;
  model: string | null;
  tokens: number | null;
  cost: number | null;
  latency_ms: number | null;
}

const typeIcons: Record<string, string> = {
  llm_call: '●',
  tool_call: '⬡',
  tool_result: '◆',
  decision_point: '◇',
};

export default function NarrativeView({ nodes }: { nodes: GraphNode[] }) {
  const [segments] = useState<NarrativeSegment[]>(() => generateNarrative(nodes));

  if (segments.length === 0) {
    return <div className="text-zinc-500 text-sm">No data to narrate.</div>;
  }

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">Narrative</h3>
      <div className="space-y-2 bg-zinc-900 rounded-lg p-4 border border-zinc-800">
        {segments.map((seg, i) => (
          <div key={seg.nodeId} className="flex items-start gap-3 text-sm">
            <span className="text-zinc-500 mt-0.5 w-4 text-center">
              {typeIcons[seg.nodeType] || '•'}
            </span>
            <span className="text-zinc-300">{seg.text}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Verify it builds**

```bash
cd ~/projects-wsl/corvade/dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
cd ~/projects-wsl/corvade
git add dashboard/
git commit -m "feat: add NarrativeView with template-based narrative engine"
```

---

## Task 18: Embed Dashboard in Go Binary

**Files:**
- Modify: `internal/api/server.go`
- Create: `internal/api/embed.go`
- Modify: `Makefile`

- [ ] **Step 1: Create embed file**

`internal/api/embed.go`:

```go
package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dashboard_dist
var dashboardFS embed.FS

func DashboardHandler() http.Handler {
	dist, err := fs.Sub(dashboardFS, "dashboard_dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(dist))
}
```

- [ ] **Step 2: Update Makefile for full build**

```makefile
.PHONY: build build-dashboard test run clean dev

build: build-dashboard
	mkdir -p internal/api/dashboard_dist
	cp -r dashboard/out/* internal/api/dashboard_dist/
	go build -o bin/corvade ./cmd/corvade

build-dashboard:
	cd dashboard && npm run build

test:
	go test ./... -v

run:
	go run ./cmd/corvade

dev:
	go run ./cmd/corvade start --dev

clean:
	rm -rf bin/ internal/api/dashboard_dist/ dashboard/out/ dashboard/.next/
```

- [ ] **Step 3: Update API server to serve dashboard**

In `internal/api/server.go`, add a fallback handler that serves the embedded dashboard for non-API routes:

```go
func NewAPIServer(store *capture.Store, hub *Hub, port int, devMode bool) *APIServer {
	mux := http.NewServeMux()
	RegisterRoutes(mux, store)

	if devMode {
		// Reverse proxy to Next.js dev server
		// (implement later when --dev flag is wired)
	} else {
		mux.Handle("/", DashboardHandler())
	}

	return &APIServer{store: store, hub: hub, mux: mux, port: port}
}
```

- [ ] **Step 4: Update next.config to enable static export**

`dashboard/next.config.ts`:

```typescript
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'export',
};

export default nextConfig;
```

- [ ] **Step 5: Build and verify full binary**

```bash
make build
ls -la bin/corvade
```

Expected: single binary that includes the dashboard

- [ ] **Step 6: Commit**

```bash
git add internal/api/embed.go Makefile dashboard/next.config.ts
git commit -m "feat: embed Next.js dashboard in Go binary via go:embed"
```

---

## Task 19: Integration Test — Full Pipeline

**Files:**
- Create: `test/integration_test.go`

- [ ] **Step 1: Write end-to-end integration test**

Tests the full pipeline: send a request through the proxy → verify it's captured → verify API returns it → verify topology is built.

```go
package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/api"
	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy"
)

func TestFullPipeline(t *testing.T) {
	// Mock upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"chatcmpl-test","model":"gpt-4o","choices":[{"message":{"content":"response"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer upstream.Close()

	// Setup
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := capture.NewStore(dbPath)
	defer store.Close()

	hub := api.NewHub()
	go hub.Run()

	calc := cost.NewCalculator(nil)
	proxySrv := proxy.NewServer(store, calc, hub)
	proxySrv.SetUpstreamURL("openai", upstream.URL)

	// Send request through proxy
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"test"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Corvade-Agent", "integration-test")
	w := httptest.NewRecorder()

	proxySrv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("proxy returned %d", w.Code)
	}

	// Wait for async capture
	time.Sleep(100 * time.Millisecond)

	// Verify via API
	apiMux := http.NewServeMux()
	api.RegisterRoutes(apiMux, store)

	apiReq := httptest.NewRequest("GET", "/api/traces?limit=10", nil)
	apiW := httptest.NewRecorder()
	apiMux.ServeHTTP(apiW, apiReq)

	var traces []capture.Trace
	respBody, _ := io.ReadAll(apiW.Body)
	json.Unmarshal(respBody, &traces)

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace via API, got %d", len(traces))
	}
	if traces[0].Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", traces[0].Model)
	}
	if traces[0].Agent == nil || *traces[0].Agent != "integration-test" {
		t.Error("agent not captured via API")
	}
	if traces[0].Cost == nil || *traces[0].Cost == 0 {
		t.Error("cost not calculated")
	}
}
```

- [ ] **Step 2: Run integration test**

```bash
go test ./test/ -v
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add test/
git commit -m "test: add full pipeline integration test (proxy → capture → API)"
```

---

## Task 20: Remaining Test Fixtures & Demo Mode

**Files:**
- Create: remaining `testdata/` fixtures
- Modify: `internal/cli/start.go` to support `--demo`

- [ ] **Step 1: Create remaining fixtures**

Create `testdata/single-llm-call.json`, `testdata/multi-agent-handoff.json`, `testdata/error-retry-sequence.json` with realistic trace data (following the same format as `tool-call-chain.json`).

- [ ] **Step 2: Implement demo mode in start command**

Add a function that loads all fixtures from `testdata/` (embedded via `go:embed`) and inserts them into the store when `--demo` is passed.

- [ ] **Step 3: Test demo mode**

```bash
make build
./bin/corvade start --demo
```

Expected: dashboard shows populated data from fixtures

- [ ] **Step 4: Commit**

```bash
git add testdata/ internal/cli/start.go
git commit -m "feat: add demo mode with fixture data and remaining test fixtures"
```

---

## Summary

| Task | Description | Key Files |
|------|------------|-----------|
| 1 | Project scaffolding | go.mod, main.go, Makefile |
| 2 | Config package | internal/config/ |
| 3 | Data models & SQLite store | internal/capture/ |
| 4 | Cost calculator | internal/cost/ |
| 5 | Retention cleanup | internal/capture/retention.go |
| 6 | OpenAI provider adapter | internal/proxy/providers/openai.go |
| 7 | Anthropic provider adapter | internal/proxy/providers/anthropic.go |
| 8 | HTTP reverse proxy | internal/proxy/ |
| 9 | WebSocket event hub | internal/api/ws.go |
| 10 | REST API handlers | internal/api/ |
| 11 | CLI — start command | internal/cli/start.go |
| 12 | CLI — tail, doctor, sessions, inspect | internal/cli/ |
| 13 | Topology inference engine | internal/topology/ |
| 14 | Dashboard scaffolding | dashboard/ |
| 15 | Timeline component | dashboard/src/components/Timeline.tsx |
| 16 | Topology Graph component | dashboard/src/components/TopologyGraph.tsx |
| 17 | Narrative View component | dashboard/src/components/NarrativeView.tsx |
| 18 | Embed dashboard in binary | internal/api/embed.go, Makefile |
| 19 | Integration test | test/integration_test.go |
| 20 | Fixtures & demo mode | testdata/, --demo flag |

**Not included in this plan (deferred to fast-follow):**
- CLI export command (straightforward once API is done)
- CLI clear/config commands (trivial)
- Minimal SDK (TypeScript/Python — separate repos, independent work)

---

## Review Fixes — Additional Tasks

The following tasks address critical gaps found during plan review.

---

## Task 21: SSE Streaming Passthrough

**Critical:** Most LLM usage is streaming-by-default. Without this, the proxy breaks real agent workloads.

**Files:**
- Modify: `internal/proxy/server.go`
- Create: `internal/proxy/stream.go`
- Test: `internal/proxy/stream_test.go`

- [ ] **Step 1: Write streaming proxy test**

```go
package proxy

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
)

func TestProxyStreamsSSE(t *testing.T) {
	// Mock upstream that sends SSE chunks
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to be a Flusher")
		}

		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" world"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer upstream.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := capture.NewStore(dbPath)
	defer store.Close()

	srv := NewServer(store, cost.NewCalculator(nil), nil)
	srv.SetUpstreamURL("openai", upstream.URL)

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Verify response is SSE
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	// Verify all chunks were forwarded
	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	chunkCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			chunkCount++
		}
	}
	if chunkCount < 3 {
		t.Errorf("expected at least 3 data chunks, got %d", chunkCount)
	}

	// Wait for async capture
	time.Sleep(200 * time.Millisecond)

	// Verify trace was captured with assembled response
	traces, _ := store.ListTraces(capture.TraceFilter{Limit: 10})
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].Response == nil {
		t.Error("expected assembled response to be captured")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/proxy/ -run TestProxyStreamsSSE -v
```

- [ ] **Step 3: Implement SSE streaming**

`internal/proxy/stream.go`:

```go
package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// streamSSE forwards an SSE response from upstream to the client in real-time,
// while assembling the full response for capture. Returns the assembled content
// and usage info extracted from the final chunk.
func streamSSE(upstream io.Reader, w http.ResponseWriter) (assembled string, usage *streamUsage, err error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback: read everything and write at once
		data, err := io.ReadAll(upstream)
		if err != nil {
			return "", nil, err
		}
		w.Write(data)
		return string(data), nil, nil
	}

	var contentBuf bytes.Buffer
	var lastUsage *streamUsage
	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()

		// Forward line immediately
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()

		// Parse SSE data lines for content assembly
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}
			contentBuf.WriteString(line)
			contentBuf.WriteString("\n")

			// Try to extract usage from final chunk
			var chunk struct {
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					CachedTokens     int `json:"cached_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil && chunk.Usage != nil {
				lastUsage = &streamUsage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					CachedTokens:     chunk.Usage.CachedTokens,
				}
			}
		}
	}

	// Write final newline
	fmt.Fprintf(w, "\n")
	flusher.Flush()

	return contentBuf.String(), lastUsage, scanner.Err()
}

type streamUsage struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
}
```

- [ ] **Step 4: Update proxy server to detect streaming and use streamSSE**

In `internal/proxy/server.go`, modify `handleProxy` to check if the response is SSE:

```go
// After getting the upstream response, check content type
isSSE := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

// Write response headers
for k, v := range resp.Header {
	for _, vv := range v {
		w.Header().Add(k, vv)
	}
}
w.WriteHeader(resp.StatusCode)

var respBodyStr string
var tokenInfo *providers.ResponseInfo

if isSSE {
	// Stream SSE chunks through to client in real-time
	assembled, usage, err := streamSSE(resp.Body, w)
	if err != nil {
		log.Printf("warning: SSE streaming error: %v", err)
	}
	respBodyStr = assembled
	if usage != nil {
		tokenInfo = &providers.ResponseInfo{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			CachedTokens:     usage.CachedTokens,
		}
	}
} else {
	// Non-streaming: read full response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("warning: failed to read upstream response: %v", err)
	}
	w.Write(respBody)
	respBodyStr = string(respBody)

	switch provider {
	case "openai":
		tokenInfo, _ = providers.OpenAIParseResponse(respBody)
	case "anthropic":
		tokenInfo, _ = providers.AnthropicParseResponse(respBody)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/proxy/ -v
```

Expected: all tests PASS including streaming test

- [ ] **Step 6: Commit**

```bash
git add internal/proxy/stream.go internal/proxy/stream_test.go internal/proxy/server.go
git commit -m "feat: add SSE streaming passthrough with real-time forwarding and response assembly"
```

---

## Task 22: WebSocket Upgrade Handler

**Critical:** Without this, the dashboard can't receive live updates and `corvade tail` can't connect.

**Files:**
- Modify: `internal/api/ws.go`
- Modify: `internal/api/routes.go`

- [ ] **Step 1: Add WebSocket upgrade handler**

Add to `internal/api/ws.go`:

```go
import (
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Local-only, trusted
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:  h,
		send: make(chan []byte, 256),
	}
	h.Register(client)

	// Writer goroutine
	go func() {
		defer func() {
			conn.Close()
		}()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()

	// Reader goroutine (just drain — we don't expect client messages)
	go func() {
		defer func() {
			h.Unregister(client)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
```

- [ ] **Step 2: Register /ws route**

In `internal/api/routes.go`, add:

```go
func RegisterRoutes(mux *http.ServeMux, store *capture.Store, hub *Hub) {
	// ... existing routes ...
	mux.HandleFunc("/ws", hub.HandleWebSocket)
}
```

Update the `RegisterRoutes` signature to accept `*Hub` and update all callers.

- [ ] **Step 3: Verify tail command can connect**

Start corvade and run `corvade tail` in a separate terminal. Verify it connects without error.

- [ ] **Step 4: Commit**

```bash
git add internal/api/ws.go internal/api/routes.go internal/api/server.go
git commit -m "feat: add WebSocket upgrade handler for live dashboard updates and tail command"
```

---

## Task 23: Session Diff — API Handler & Component

**Files:**
- Create: `internal/api/handlers/diff.go`
- Create: `dashboard/src/components/SessionDiff.tsx`
- Create: `dashboard/src/app/diff/page.tsx`
- Modify: `internal/api/routes.go`

- [ ] **Step 1: Implement structural diff algorithm in Go**

`internal/api/handlers/diff.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/corvade/corvade/internal/capture"
)

type DiffHandler struct {
	store *capture.Store
}

func NewDiffHandler(store *capture.Store) *DiffHandler {
	return &DiffHandler{store: store}
}

type AlignedPair struct {
	Left       *capture.GraphNode `json:"left"`
	Right      *capture.GraphNode `json:"right"`
	MatchScore float64            `json:"match_score"`
}

type DiffResponse struct {
	AlignedNodes     []AlignedPair       `json:"aligned_nodes"`
	LeftOnly         []capture.GraphNode  `json:"left_only"`
	RightOnly        []capture.GraphNode  `json:"right_only"`
	DivergencePoints []capture.GraphNode  `json:"divergence_points"`
}

func (h *DiffHandler) Diff(w http.ResponseWriter, r *http.Request, id1, id2 string) {
	nodesA, _, err := h.store.GetSessionGraph(id1)
	if err != nil {
		http.Error(w, "session not found: "+id1, http.StatusNotFound)
		return
	}
	nodesB, _, err := h.store.GetSessionGraph(id2)
	if err != nil {
		http.Error(w, "session not found: "+id2, http.StatusNotFound)
		return
	}

	result := structuralDiff(nodesA, nodesB)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func structuralDiff(a, b []capture.GraphNode) DiffResponse {
	var aligned []AlignedPair
	matchedB := make(map[int]bool)

	for _, nodeA := range a {
		bestMatch := -1
		bestScore := 0.0

		for j, nodeB := range b {
			if matchedB[j] {
				continue
			}
			score := matchNodes(nodeA, nodeB)
			if score > bestScore && score >= 0.5 {
				bestScore = score
				bestMatch = j
			}
		}

		if bestMatch >= 0 {
			matchedB[bestMatch] = true
			nA := nodeA
			nB := b[bestMatch]
			aligned = append(aligned, AlignedPair{Left: &nA, Right: &nB, MatchScore: bestScore})
		}
	}

	// Unmatched nodes
	var leftOnly, rightOnly []capture.GraphNode
	for _, nA := range a {
		found := false
		for _, pair := range aligned {
			if pair.Left.ID == nA.ID {
				found = true
				break
			}
		}
		if !found {
			leftOnly = append(leftOnly, nA)
		}
	}
	for j, nB := range b {
		if !matchedB[j] {
			rightOnly = append(rightOnly, nB)
		}
	}

	// Divergence points: aligned pairs with low match scores
	var divergence []capture.GraphNode
	for _, pair := range aligned {
		if pair.MatchScore < 0.8 {
			divergence = append(divergence, *pair.Left)
		}
	}

	return DiffResponse{
		AlignedNodes:     aligned,
		LeftOnly:         leftOnly,
		RightOnly:        rightOnly,
		DivergencePoints: divergence,
	}
}

func matchNodes(a, b capture.GraphNode) float64 {
	score := 0.0
	checks := 0.0

	// Same agent + step label
	if a.Agent != nil && b.Agent != nil && *a.Agent == *b.Agent {
		score += 1.0
	}
	checks += 1.0

	if a.Step != nil && b.Step != nil && *a.Step == *b.Step {
		score += 1.0
	}
	checks += 1.0

	// Same type
	if a.Type == b.Type {
		score += 0.5
	}
	checks += 0.5

	// Same model
	if a.Model != nil && b.Model != nil && *a.Model == *b.Model {
		score += 0.5
	}
	checks += 0.5

	// Jaccard similarity on context snapshot (tokenized)
	if a.ContextSnapshot != nil && b.ContextSnapshot != nil {
		j := jaccard(tokenize(*a.ContextSnapshot), tokenize(*b.ContextSnapshot))
		if j >= 0.7 {
			score += 1.0
		}
	}
	checks += 1.0

	if checks == 0 {
		return 0
	}
	return score / checks
}

func tokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	for _, word := range strings.Fields(s) {
		tokens[strings.ToLower(word)] = true
	}
	return tokens
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
```

- [ ] **Step 2: Register diff route**

Add to `internal/api/routes.go` inside the `/api/sessions/` handler:

```go
if len(parts) == 3 && parts[1] == "diff" {
	diff := handlers.NewDiffHandler(store)
	diff.Diff(w, r, parts[0], parts[2])
	return
}
```

- [ ] **Step 3: Build SessionDiff component**

`dashboard/src/components/SessionDiff.tsx`:

```tsx
'use client';

import { useEffect, useState } from 'react';

interface GraphNode {
  id: string;
  type: string;
  agent: string | null;
  step: string | null;
  model: string | null;
}

interface AlignedPair {
  left: GraphNode;
  right: GraphNode;
  match_score: number;
}

interface DiffResult {
  aligned_nodes: AlignedPair[];
  left_only: GraphNode[];
  right_only: GraphNode[];
  divergence_points: GraphNode[];
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

export default function SessionDiff({ sessionA, sessionB }: { sessionA: string; sessionB: string }) {
  const [diff, setDiff] = useState<DiffResult | null>(null);

  useEffect(() => {
    fetch(`${API_BASE}/api/sessions/${sessionA}/diff/${sessionB}`)
      .then(r => r.json())
      .then(setDiff)
      .catch(console.error);
  }, [sessionA, sessionB]);

  if (!diff) return <div className="text-zinc-500">Loading diff...</div>;

  return (
    <div className="space-y-6">
      {/* Aligned nodes */}
      <div>
        <h3 className="text-sm font-semibold text-zinc-400 mb-2">Matched ({diff.aligned_nodes?.length || 0})</h3>
        <div className="space-y-2">
          {(diff.aligned_nodes || []).map((pair, i) => (
            <div key={i} className="flex gap-4 text-sm">
              <div className="flex-1 bg-zinc-900 rounded p-2 border border-zinc-800">
                <span className="text-zinc-300">{pair.left.model}</span>
                <span className="text-zinc-500 ml-2">{pair.left.step || pair.left.type}</span>
              </div>
              <div className="flex items-center text-zinc-500 text-xs">
                {(pair.match_score * 100).toFixed(0)}%
              </div>
              <div className="flex-1 bg-zinc-900 rounded p-2 border border-zinc-800">
                <span className="text-zinc-300">{pair.right.model}</span>
                <span className="text-zinc-500 ml-2">{pair.right.step || pair.right.type}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Left only */}
      {diff.left_only?.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-red-400 mb-2">Only in Session A ({diff.left_only.length})</h3>
          {diff.left_only.map(n => (
            <div key={n.id} className="text-sm text-red-300 bg-red-950/30 rounded p-2 border border-red-900/50 mb-1">
              {n.model} — {n.step || n.type}
            </div>
          ))}
        </div>
      )}

      {/* Right only */}
      {diff.right_only?.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-green-400 mb-2">Only in Session B ({diff.right_only.length})</h3>
          {diff.right_only.map(n => (
            <div key={n.id} className="text-sm text-green-300 bg-green-950/30 rounded p-2 border border-green-900/50 mb-1">
              {n.model} — {n.step || n.type}
            </div>
          ))}
        </div>
      )}

      {/* Divergence points */}
      {diff.divergence_points?.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-yellow-400 mb-2">Divergence Points ({diff.divergence_points.length})</h3>
          {diff.divergence_points.map(n => (
            <div key={n.id} className="text-sm text-yellow-300 bg-yellow-950/30 rounded p-2 border border-yellow-900/50 mb-1">
              {n.model} — {n.step || n.type}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Create diff page**

`dashboard/src/app/diff/page.tsx`:

```tsx
'use client';

import { useState } from 'react';
import SessionDiff from '@/components/SessionDiff';

export default function DiffPage() {
  const [sessionA, setSessionA] = useState('');
  const [sessionB, setSessionB] = useState('');
  const [comparing, setComparing] = useState(false);

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <a href="/" className="text-zinc-500 hover:text-zinc-300">← Back</a>
          <h1 className="text-xl font-semibold">Session Diff</h1>
        </div>

        <div className="flex gap-3 mb-6">
          <input
            type="text"
            placeholder="Session A ID..."
            className="bg-zinc-800 border border-zinc-700 rounded px-3 py-1.5 text-sm text-zinc-200 flex-1"
            value={sessionA}
            onChange={(e) => setSessionA(e.target.value)}
          />
          <span className="text-zinc-500 self-center">vs</span>
          <input
            type="text"
            placeholder="Session B ID..."
            className="bg-zinc-800 border border-zinc-700 rounded px-3 py-1.5 text-sm text-zinc-200 flex-1"
            value={sessionB}
            onChange={(e) => setSessionB(e.target.value)}
          />
          <button
            className="bg-blue-600 hover:bg-blue-500 text-white px-4 py-1.5 rounded text-sm"
            onClick={() => setComparing(true)}
            disabled={!sessionA || !sessionB}
          >
            Compare
          </button>
        </div>

        {comparing && sessionA && sessionB && (
          <SessionDiff sessionA={sessionA} sessionB={sessionB} />
        )}
      </div>
    </main>
  );
}
```

- [ ] **Step 5: Verify it builds**

```bash
cd ~/projects-wsl/corvade/dashboard && npm run build
```

- [ ] **Step 6: Commit**

```bash
cd ~/projects-wsl/corvade
git add internal/api/handlers/diff.go dashboard/src/components/SessionDiff.tsx dashboard/src/app/diff/page.tsx internal/api/routes.go
git commit -m "feat: add session diff with structural matching, API handler, and dashboard component"
```

---

## Task 24: Detail Inspector Component

**Files:**
- Create: `dashboard/src/components/DetailInspector.tsx`

- [ ] **Step 1: Build DetailInspector**

```tsx
'use client';

import { useEffect, useState } from 'react';
import { fetchTrace } from '@/lib/api';

interface Trace {
  id: string;
  model: string;
  provider: string;
  agent: string | null;
  step: string | null;
  request: string;
  response: string | null;
  status_code: number;
  tokens_prompt: number | null;
  tokens_completion: number | null;
  tokens_cached: number | null;
  cost: number | null;
  latency_ms: number | null;
  ttft_ms: number | null;
  created_at: string;
}

export default function DetailInspector({ traceId, onClose }: { traceId: string; onClose: () => void }) {
  const [trace, setTrace] = useState<Trace | null>(null);
  const [activeTab, setActiveTab] = useState<'request' | 'response'>('request');

  useEffect(() => {
    fetchTrace(traceId).then(setTrace).catch(console.error);
  }, [traceId]);

  if (!trace) return <div className="text-zinc-500">Loading...</div>;

  const formatJSON = (s: string | null) => {
    if (!s) return 'null';
    try { return JSON.stringify(JSON.parse(s), null, 2); } catch { return s; }
  };

  return (
    <div className="bg-zinc-900 rounded-lg border border-zinc-800 p-4 space-y-4">
      <div className="flex justify-between items-center">
        <div>
          <span className="text-zinc-200 font-semibold">{trace.model}</span>
          <span className="text-zinc-500 ml-2 text-sm">{trace.provider}</span>
          <span className="text-zinc-500 ml-2 text-sm">ID: {trace.id.slice(0, 12)}</span>
        </div>
        <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300 text-sm">✕ Close</button>
      </div>

      {/* Metrics bar */}
      <div className="flex gap-6 text-sm">
        <div>
          <span className="text-zinc-500">Tokens:</span>
          <span className="text-zinc-300 ml-1">
            {trace.tokens_prompt || 0} in / {trace.tokens_completion || 0} out
            {trace.tokens_cached ? ` (${trace.tokens_cached} cached)` : ''}
          </span>
        </div>
        <div>
          <span className="text-zinc-500">Cost:</span>
          <span className="text-zinc-300 ml-1">{trace.cost != null ? `$${trace.cost.toFixed(4)}` : '-'}</span>
        </div>
        <div>
          <span className="text-zinc-500">Latency:</span>
          <span className="text-zinc-300 ml-1">{trace.latency_ms ? `${(trace.latency_ms / 1000).toFixed(2)}s` : '-'}</span>
        </div>
        {trace.ttft_ms && (
          <div>
            <span className="text-zinc-500">TTFT:</span>
            <span className="text-zinc-300 ml-1">{(trace.ttft_ms / 1000).toFixed(2)}s</span>
          </div>
        )}
        <div>
          <span className="text-zinc-500">Status:</span>
          <span className={`ml-1 ${trace.status_code >= 400 ? 'text-red-400' : 'text-green-400'}`}>
            {trace.status_code}
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-zinc-800">
        <button
          className={`px-3 py-1.5 text-sm ${activeTab === 'request' ? 'text-zinc-200 border-b-2 border-blue-500' : 'text-zinc-500'}`}
          onClick={() => setActiveTab('request')}
        >Request</button>
        <button
          className={`px-3 py-1.5 text-sm ${activeTab === 'response' ? 'text-zinc-200 border-b-2 border-blue-500' : 'text-zinc-500'}`}
          onClick={() => setActiveTab('response')}
        >Response</button>
      </div>

      {/* JSON viewer */}
      <pre className="bg-zinc-950 rounded p-3 text-xs text-zinc-300 overflow-auto max-h-96 font-mono">
        {activeTab === 'request' ? formatJSON(trace.request) : formatJSON(trace.response)}
      </pre>
    </div>
  );
}
```

- [ ] **Step 2: Wire into Timeline (add click handler)**

Update `dashboard/src/components/Timeline.tsx` to open DetailInspector when a trace row is clicked.

- [ ] **Step 3: Verify it builds**

```bash
cd ~/projects-wsl/corvade/dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
cd ~/projects-wsl/corvade
git add dashboard/src/components/DetailInspector.tsx dashboard/src/components/Timeline.tsx
git commit -m "feat: add DetailInspector component with request/response viewer and token breakdown"
```

---

## Task 25: SQL Aggregation for Stats & Bug Fixes

Fixes the Stats handler to use SQL aggregation instead of loading all traces into memory, plus miscellaneous fixes from review.

**Files:**
- Modify: `internal/capture/store.go` — add `GetStats` method, fix `InsertSession` end_time bug
- Modify: `internal/api/handlers/export.go` — use SQL aggregation
- Modify: `internal/topology/fingerprint.go` — remove redundant `min` function
- Create: `internal/api/dashboard_dist/.gitkeep` — placeholder for embed

- [ ] **Step 1: Add GetStats to store**

```go
type Stats struct {
	TotalTraces   int                `json:"total_traces"`
	TotalSessions int                `json:"total_sessions"`
	TotalCost     float64            `json:"total_cost"`
	TotalTokens   int                `json:"total_tokens"`
	ByModel       map[string]int     `json:"by_model"`
	ByAgent       map[string]int     `json:"by_agent"`
}

func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{ByModel: make(map[string]int), ByAgent: make(map[string]int)}

	s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(cost), 0), COALESCE(SUM(COALESCE(tokens_prompt,0)+COALESCE(tokens_completion,0)), 0) FROM traces").
		Scan(&stats.TotalTraces, &stats.TotalCost, &stats.TotalTokens)

	s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&stats.TotalSessions)

	rows, _ := s.db.Query("SELECT model, COUNT(*) FROM traces GROUP BY model")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var count int
			rows.Scan(&model, &count)
			stats.ByModel[model] = count
		}
	}

	rows2, _ := s.db.Query("SELECT agent, COUNT(*) FROM traces WHERE agent IS NOT NULL GROUP BY agent")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var agent string
			var count int
			rows2.Scan(&agent, &count)
			stats.ByAgent[agent] = count
		}
	}

	return stats, nil
}
```

- [ ] **Step 2: Fix InsertSession to use sess.EndTime**

- [ ] **Step 3: Remove custom `min` from fingerprint.go (Go 1.22 built-in)**

- [ ] **Step 4: Create placeholder for embed**

```bash
mkdir -p internal/api/dashboard_dist
touch internal/api/dashboard_dist/.gitkeep
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/capture/store.go internal/api/handlers/export.go internal/topology/fingerprint.go internal/api/dashboard_dist/.gitkeep
git commit -m "fix: SQL aggregation for stats, InsertSession end_time, remove redundant min, add embed placeholder"
```

---

---

## Task 26: Narrative REST API Endpoints

**Files:**
- Modify: `internal/api/handlers/topology.go` — add narrative and enhance handlers
- Modify: `internal/api/routes.go` — register narrative routes

- [ ] **Step 1: Add server-side narrative template engine in Go**

Add to `internal/api/handlers/topology.go`:

```go
type NarrativeSegment struct {
	NodeID   string `json:"node_id"`
	Text     string `json:"text"`
	NodeType string `json:"node_type"`
}

type NarrativeResponse struct {
	Segments []NarrativeSegment `json:"segments"`
}

func (h *TopologyHandler) GetNarrative(w http.ResponseWriter, r *http.Request, sessionID string) {
	nodes, _, err := h.store.GetSessionGraph(sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	segments := make([]NarrativeSegment, 0, len(nodes))
	for _, n := range nodes {
		text := renderNarrativeNode(n)
		segments = append(segments, NarrativeSegment{
			NodeID:   n.ID,
			Text:     text,
			NodeType: n.Type,
		})
	}

	format := r.URL.Query().Get("format")
	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown")
		for _, seg := range segments {
			fmt.Fprintf(w, "- %s\n", seg.Text)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NarrativeResponse{Segments: segments})
}

func renderNarrativeNode(n capture.GraphNode) string {
	agent := "The agent"
	if n.Agent != nil {
		agent = *n.Agent
	}
	model := "the LLM"
	if n.Model != nil {
		model = *n.Model
	}
	step := "process the request"
	if n.Step != nil {
		step = *n.Step
	}

	costStr := ""
	if n.Cost != nil {
		costStr = fmt.Sprintf(" (cost: $%.4f)", *n.Cost)
	}
	latStr := ""
	if n.LatencyMS != nil {
		latStr = fmt.Sprintf(" in %.1fs", float64(*n.LatencyMS)/1000)
	}

	switch n.Type {
	case "llm_call":
		return fmt.Sprintf("%s asked %s to %s%s%s.", agent, model, step, latStr, costStr)
	case "tool_call":
		return fmt.Sprintf("It called %s%s.", step, latStr)
	case "tool_result":
		return fmt.Sprintf("It received the result from %s.", step)
	case "decision_point":
		return fmt.Sprintf("It decided to %s.", step)
	default:
		return fmt.Sprintf("%s performed %s%s%s.", agent, n.Type, latStr, costStr)
	}
}
```

- [ ] **Step 2: Add LLM enhance endpoint**

```go
type EnhanceRequest struct {
	Model string `json:"model,omitempty"` // defaults to first model seen in session
}

type EnhanceResponse struct {
	Segments   []NarrativeSegment `json:"segments"`
	ModelUsed  string             `json:"model_used"`
	TokensUsed int                `json:"tokens_used"`
	Cost       float64            `json:"cost"`
}

func (h *TopologyHandler) EnhanceNarrative(w http.ResponseWriter, r *http.Request, sessionID string) {
	nodes, edges, err := h.store.GetSessionGraph(sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Build context from graph data
	var context strings.Builder
	context.WriteString("Analyze this AI agent execution trace and explain what happened in plain English:\n\n")
	for _, n := range nodes {
		context.WriteString(fmt.Sprintf("- [%s] %s", n.Type, renderNarrativeNode(n)))
		context.WriteString("\n")
	}
	context.WriteString(fmt.Sprintf("\nThe trace has %d nodes and %d edges.\n", len(nodes), len(edges)))
	context.WriteString("Provide a clear, concise narrative explanation of what the agent did and why.")

	// Determine model — use first model from the session's traces
	model := "gpt-4o-mini" // cheap default
	for _, n := range nodes {
		if n.Model != nil {
			model = *n.Model
			break
		}
	}

	// Send request through the proxy to the user's own LLM
	// For v1, return an error explaining this requires manual setup
	// Full implementation routes through localhost:4400
	http.Error(w, "LLM-enhanced narrative requires proxy configuration. Use the template narrative endpoint instead.", http.StatusNotImplemented)
}
```

- [ ] **Step 3: Register narrative routes**

In `internal/api/routes.go`, inside the `/api/sessions/` handler:

```go
if len(parts) == 2 && parts[1] == "narrative" {
	if r.Method == "POST" {
		// Check if it's the enhance sub-path
		// Actually route: /api/sessions/:id/narrative/enhance is 3 parts
		topology.GetNarrative(w, r, parts[0])
		return
	}
	topology.GetNarrative(w, r, parts[0])
	return
}
if len(parts) == 3 && parts[1] == "narrative" && parts[2] == "enhance" {
	topology.EnhanceNarrative(w, r, parts[0])
	return
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/handlers/topology.go internal/api/routes.go
git commit -m "feat: add narrative REST API endpoints with template rendering and enhance stub"
```

---

## Corrections to Earlier Tasks

**Task 25, Step 1 — Add time-range filtering to GetStats:**

The `GetStats` method should accept optional `from`/`to` parameters:

```go
func (s *Store) GetStats(from, to *time.Time) (*Stats, error) {
	stats := &Stats{ByModel: make(map[string]int), ByAgent: make(map[string]int)}

	whereClause := ""
	args := []interface{}{}
	if from != nil {
		whereClause += " AND created_at >= ?"
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if to != nil {
		whereClause += " AND created_at <= ?"
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}

	query := "SELECT COUNT(*), COALESCE(SUM(cost), 0), COALESCE(SUM(COALESCE(tokens_prompt,0)+COALESCE(tokens_completion,0)), 0) FROM traces WHERE 1=1" + whereClause
	s.db.QueryRow(query, args...).Scan(&stats.TotalTraces, &stats.TotalCost, &stats.TotalTokens)

	s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&stats.TotalSessions)

	modelQuery := "SELECT model, COUNT(*) FROM traces WHERE 1=1" + whereClause + " GROUP BY model"
	rows, _ := s.db.Query(modelQuery, args...)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var count int
			rows.Scan(&model, &count)
			stats.ByModel[model] = count
		}
	}

	agentQuery := "SELECT agent, COUNT(*) FROM traces WHERE agent IS NOT NULL" + whereClause + " GROUP BY agent"
	rows2, _ := s.db.Query(agentQuery, args...)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var agent string
			var count int
			rows2.Scan(&agent, &count)
			stats.ByAgent[agent] = count
		}
	}

	return stats, nil
}
```

The `export.go` Stats handler parses `from`/`to` from query parameters and passes them to `GetStats`.

**Task 25, Step 2 — Fix InsertSession end_time:**

Replace the `nil` in the InsertSession Exec call with:

```go
var endTimeStr *string
if sess.EndTime != nil {
	s := sess.EndTime.UTC().Format(time.RFC3339Nano)
	endTimeStr = &s
}
// Then use endTimeStr instead of nil in the Exec call
```

**Task 25, Step 3 — Remove custom min:**

Delete the `min` function at the bottom of `internal/topology/fingerprint.go`. Go 1.22+ provides `min` as a built-in.

**Task 23 — Note on diff algorithm:**

The structural diff uses greedy best-match rather than LCS as specified in the design spec. This is a deliberate v1 simplification — greedy matching handles the primary use cases (same agent with different models or prompt changes) without the complexity of LCS alignment. LCS can be added in a future iteration if needed.

---

## Updated Summary

| Task | Description | Key Files |
|------|------------|-----------|
| 1-20 | (original tasks — see above) | |
| 21 | SSE streaming passthrough | internal/proxy/stream.go |
| 22 | WebSocket upgrade handler | internal/api/ws.go |
| 23 | Session Diff (API + component) | internal/api/handlers/diff.go, SessionDiff.tsx |
| 24 | Detail Inspector component | DetailInspector.tsx |
| 25 | SQL aggregation & bug fixes | store.go, export.go, fingerprint.go |
| 26 | Narrative REST API endpoints | internal/api/handlers/topology.go |
