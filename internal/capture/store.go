package capture

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid/v2"
)

// NewULID generates a new ULID string.
func NewULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// TraceFilter specifies optional filters for listing traces.
type TraceFilter struct {
	SessionID *string
	Provider  *string
	Model     *string
	Agent     *string
	Search    *string
	Limit     *int
	Offset    *int
}

// SessionFilter specifies optional filters for listing sessions.
type SessionFilter struct {
	Agent  *string
	Limit  *int
	Offset *int
}

// Stats holds aggregate statistics across traces.
type Stats struct {
	TraceCount  int     `json:"trace_count"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens int     `json:"total_tokens"`
}

// Store is a SQLite-backed storage for traces, sessions, and graph data.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS traces (
	id TEXT PRIMARY KEY,
	session_id TEXT,
	agent TEXT,
	step TEXT,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	request TEXT NOT NULL,
	response TEXT,
	status_code INTEGER NOT NULL,
	tokens_prompt INTEGER,
	tokens_completion INTEGER,
	tokens_cached INTEGER,
	cost REAL,
	latency_ms INTEGER,
	ttft_ms INTEGER,
	api_key_hash TEXT,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(session_id);
CREATE INDEX IF NOT EXISTS idx_traces_provider ON traces(provider);
CREATE INDEX IF NOT EXISTS idx_traces_model ON traces(model);
CREATE INDEX IF NOT EXISTS idx_traces_created ON traces(created_at);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	agent TEXT,
	start_time TEXT NOT NULL,
	end_time TEXT,
	trace_count INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	total_cost REAL NOT NULL DEFAULT 0,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at);

CREATE TABLE IF NOT EXISTS graph_nodes (
	id TEXT PRIMARY KEY,
	trace_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	type TEXT NOT NULL,
	agent TEXT,
	step TEXT,
	model TEXT,
	tokens INTEGER,
	cost REAL,
	latency_ms INTEGER,
	context_snapshot TEXT,
	confidence REAL NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_graph_nodes_session ON graph_nodes(session_id);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_trace ON graph_nodes(trace_id);

CREATE TABLE IF NOT EXISTS graph_edges (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	from_node TEXT NOT NULL,
	to_node TEXT NOT NULL,
	type TEXT NOT NULL,
	confidence REAL NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_session ON graph_edges(session_id);
`

// NewStore opens (or creates) a SQLite database at path, enables WAL mode,
// and runs schema migration.
func NewStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("run schema migration: %w", err)
	}

	return &Store{db: db}, nil
}

// DB returns the underlying *sql.DB for use by other packages (e.g., retention).
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// InsertTrace inserts a new trace and returns its generated ID.
func (s *Store) InsertTrace(tr Trace) (string, error) {
	tr.ID = NewULID()
	tr.CreatedAt = time.Now().UTC()

	_, err := s.db.Exec(`INSERT INTO traces
		(id, session_id, agent, step, provider, model, request, response,
		 status_code, tokens_prompt, tokens_completion, tokens_cached,
		 cost, latency_ms, ttft_ms, api_key_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tr.ID, tr.SessionID, tr.Agent, tr.Step, tr.Provider, tr.Model,
		tr.Request, tr.Response, tr.StatusCode,
		tr.TokensPrompt, tr.TokensCompletion, tr.TokensCached,
		tr.Cost, tr.LatencyMS, tr.TTFTMS, tr.APIKeyHash,
		tr.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", fmt.Errorf("insert trace: %w", err)
	}
	return tr.ID, nil
}

// GetTrace retrieves a single trace by ID.
func (s *Store) GetTrace(id string) (Trace, error) {
	var tr Trace
	var createdAt string

	err := s.db.QueryRow(`SELECT
		id, session_id, agent, step, provider, model, request, response,
		status_code, tokens_prompt, tokens_completion, tokens_cached,
		cost, latency_ms, ttft_ms, api_key_hash, created_at
		FROM traces WHERE id = ?`, id).Scan(
		&tr.ID, &tr.SessionID, &tr.Agent, &tr.Step, &tr.Provider, &tr.Model,
		&tr.Request, &tr.Response, &tr.StatusCode,
		&tr.TokensPrompt, &tr.TokensCompletion, &tr.TokensCached,
		&tr.Cost, &tr.LatencyMS, &tr.TTFTMS, &tr.APIKeyHash,
		&createdAt,
	)
	if err != nil {
		return Trace{}, fmt.Errorf("get trace %s: %w", id, err)
	}

	tr.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return tr, nil
}

// ListTraces returns traces matching the given filter.
func (s *Store) ListTraces(f TraceFilter) ([]Trace, error) {
	query := `SELECT
		id, session_id, agent, step, provider, model, request, response,
		status_code, tokens_prompt, tokens_completion, tokens_cached,
		cost, latency_ms, ttft_ms, api_key_hash, created_at
		FROM traces`

	var conditions []string
	var args []interface{}

	if f.SessionID != nil {
		conditions = append(conditions, "session_id = ?")
		args = append(args, *f.SessionID)
	}
	if f.Provider != nil {
		conditions = append(conditions, "provider = ?")
		args = append(args, *f.Provider)
	}
	if f.Model != nil {
		conditions = append(conditions, "model = ?")
		args = append(args, *f.Model)
	}
	if f.Agent != nil {
		conditions = append(conditions, "agent = ?")
		args = append(args, *f.Agent)
	}
	if f.Search != nil {
		conditions = append(conditions, "(request LIKE ? OR response LIKE ? OR model LIKE ? OR agent LIKE ?)")
		pattern := "%" + *f.Search + "%"
		args = append(args, pattern, pattern, pattern, pattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if f.Limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *f.Limit)
	}
	if f.Offset != nil {
		query += fmt.Sprintf(" OFFSET %d", *f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var tr Trace
		var createdAt string
		err := rows.Scan(
			&tr.ID, &tr.SessionID, &tr.Agent, &tr.Step, &tr.Provider, &tr.Model,
			&tr.Request, &tr.Response, &tr.StatusCode,
			&tr.TokensPrompt, &tr.TokensCompletion, &tr.TokensCached,
			&tr.Cost, &tr.LatencyMS, &tr.TTFTMS, &tr.APIKeyHash,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		tr.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		traces = append(traces, tr)
	}
	return traces, rows.Err()
}

// InsertSession inserts a new session and returns its generated ID.
func (s *Store) InsertSession(sess Session) (string, error) {
	sess.ID = NewULID()
	sess.CreatedAt = time.Now().UTC()

	var endTimeStr *string
	if sess.EndTime != nil {
		st := sess.EndTime.UTC().Format(time.RFC3339Nano)
		endTimeStr = &st
	}

	_, err := s.db.Exec(`INSERT INTO sessions
		(id, agent, start_time, end_time, trace_count, total_tokens, total_cost, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.Agent,
		sess.StartTime.UTC().Format(time.RFC3339Nano),
		endTimeStr,
		sess.TraceCount, sess.TotalTokens, sess.TotalCost,
		sess.Status,
		sess.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return sess.ID, nil
}

// GetSession retrieves a single session by ID.
func (s *Store) GetSession(id string) (Session, error) {
	var sess Session
	var startTime, createdAt string
	var endTime *string

	err := s.db.QueryRow(`SELECT
		id, agent, start_time, end_time, trace_count, total_tokens, total_cost, status, created_at
		FROM sessions WHERE id = ?`, id).Scan(
		&sess.ID, &sess.Agent, &startTime, &endTime,
		&sess.TraceCount, &sess.TotalTokens, &sess.TotalCost,
		&sess.Status, &createdAt,
	)
	if err != nil {
		return Session{}, fmt.Errorf("get session %s: %w", id, err)
	}

	sess.StartTime, _ = time.Parse(time.RFC3339Nano, startTime)
	sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if endTime != nil {
		t, _ := time.Parse(time.RFC3339Nano, *endTime)
		sess.EndTime = &t
	}
	return sess, nil
}

// ListSessions returns all sessions ordered by creation time descending.
func (s *Store) ListSessions() ([]Session, error) {
	rows, err := s.db.Query(`SELECT
		id, agent, start_time, end_time, trace_count, total_tokens, total_cost, status, created_at
		FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var startTime, createdAt string
		var endTime *string

		err := rows.Scan(
			&sess.ID, &sess.Agent, &startTime, &endTime,
			&sess.TraceCount, &sess.TotalTokens, &sess.TotalCost,
			&sess.Status, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		sess.StartTime, _ = time.Parse(time.RFC3339Nano, startTime)
		sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if endTime != nil {
			t, _ := time.Parse(time.RFC3339Nano, *endTime)
			sess.EndTime = &t
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// UpdateSessionStats recalculates trace_count, total_tokens, and total_cost
// for a session from its associated traces.
func (s *Store) UpdateSessionStats(sessionID string) error {
	_, err := s.db.Exec(`UPDATE sessions SET
		trace_count = (SELECT COUNT(*) FROM traces WHERE session_id = ?),
		total_tokens = (SELECT COALESCE(SUM(COALESCE(tokens_prompt, 0) + COALESCE(tokens_completion, 0)), 0) FROM traces WHERE session_id = ?),
		total_cost = (SELECT COALESCE(SUM(COALESCE(cost, 0)), 0) FROM traces WHERE session_id = ?)
		WHERE id = ?`,
		sessionID, sessionID, sessionID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("update session stats %s: %w", sessionID, err)
	}
	return nil
}

// InsertGraphNode inserts a new graph node and returns its generated ID.
func (s *Store) InsertGraphNode(node GraphNode) (string, error) {
	node.ID = NewULID()
	node.CreatedAt = time.Now().UTC()

	_, err := s.db.Exec(`INSERT INTO graph_nodes
		(id, trace_id, session_id, type, agent, step, model, tokens, cost,
		 latency_ms, context_snapshot, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.TraceID, node.SessionID, node.Type,
		node.Agent, node.Step, node.Model, node.Tokens, node.Cost,
		node.LatencyMS, node.ContextSnapshot, node.Confidence,
		node.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", fmt.Errorf("insert graph node: %w", err)
	}
	return node.ID, nil
}

// InsertGraphEdge inserts a new graph edge and returns its generated ID.
func (s *Store) InsertGraphEdge(edge GraphEdge) (string, error) {
	edge.ID = NewULID()
	edge.CreatedAt = time.Now().UTC()

	_, err := s.db.Exec(`INSERT INTO graph_edges
		(id, session_id, from_node, to_node, type, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		edge.ID, edge.SessionID, edge.FromNode, edge.ToNode,
		edge.Type, edge.Confidence,
		edge.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", fmt.Errorf("insert graph edge: %w", err)
	}
	return edge.ID, nil
}

// GetSessionGraph returns all nodes and edges for a given session.
func (s *Store) GetSessionGraph(sessionID string) ([]GraphNode, []GraphEdge, error) {
	// Fetch nodes
	nodeRows, err := s.db.Query(`SELECT
		id, trace_id, session_id, type, agent, step, model, tokens, cost,
		latency_ms, context_snapshot, confidence, created_at
		FROM graph_nodes WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("query graph nodes: %w", err)
	}
	defer nodeRows.Close()

	var nodes []GraphNode
	for nodeRows.Next() {
		var n GraphNode
		var createdAt string
		err := nodeRows.Scan(
			&n.ID, &n.TraceID, &n.SessionID, &n.Type,
			&n.Agent, &n.Step, &n.Model, &n.Tokens, &n.Cost,
			&n.LatencyMS, &n.ContextSnapshot, &n.Confidence,
			&createdAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("scan graph node: %w", err)
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		nodes = append(nodes, n)
	}
	if err := nodeRows.Err(); err != nil {
		return nil, nil, err
	}

	// Fetch edges
	edgeRows, err := s.db.Query(`SELECT
		id, session_id, from_node, to_node, type, confidence, created_at
		FROM graph_edges WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("query graph edges: %w", err)
	}
	defer edgeRows.Close()

	var edges []GraphEdge
	for edgeRows.Next() {
		var e GraphEdge
		var createdAt string
		err := edgeRows.Scan(
			&e.ID, &e.SessionID, &e.FromNode, &e.ToNode,
			&e.Type, &e.Confidence, &createdAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("scan graph edge: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		edges = append(edges, e)
	}
	return nodes, edges, edgeRows.Err()
}

// ListSessionsFiltered returns sessions matching the given filter.
func (s *Store) ListSessionsFiltered(f SessionFilter) ([]Session, error) {
	query := `SELECT
		id, agent, start_time, end_time, trace_count, total_tokens, total_cost, status, created_at
		FROM sessions`

	var conditions []string
	var args []interface{}

	if f.Agent != nil {
		conditions = append(conditions, "agent = ?")
		args = append(args, *f.Agent)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if f.Limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *f.Limit)
	}
	if f.Offset != nil {
		query += fmt.Sprintf(" OFFSET %d", *f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var startTime, createdAt string
		var endTime *string

		err := rows.Scan(
			&sess.ID, &sess.Agent, &startTime, &endTime,
			&sess.TraceCount, &sess.TotalTokens, &sess.TotalCost,
			&sess.Status, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		sess.StartTime, _ = time.Parse(time.RFC3339Nano, startTime)
		sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if endTime != nil {
			t, _ := time.Parse(time.RFC3339Nano, *endTime)
			sess.EndTime = &t
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// GetStats returns aggregate statistics across all traces, optionally filtered
// by a time range.
func (s *Store) GetStats(from, to *time.Time) (Stats, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(cost), 0), COALESCE(SUM(COALESCE(tokens_prompt,0)+COALESCE(tokens_completion,0)), 0) FROM traces`

	var conditions []string
	var args []interface{}

	if from != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if to != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var stats Stats
	err := s.db.QueryRow(query, args...).Scan(&stats.TraceCount, &stats.TotalCost, &stats.TotalTokens)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return stats, nil
}
