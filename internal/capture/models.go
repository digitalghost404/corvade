package capture

import "time"

type Trace struct {
	ID               string    `json:"id"`
	SessionID        *string   `json:"session_id"`
	Agent            *string   `json:"agent"`
	Step             *string   `json:"step"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	Request          string    `json:"request"`
	Response         *string   `json:"response"`
	StatusCode       int       `json:"status_code"`
	TokensPrompt     *int      `json:"tokens_prompt"`
	TokensCompletion *int      `json:"tokens_completion"`
	TokensCached     *int      `json:"tokens_cached"`
	Cost             *float64  `json:"cost"`
	LatencyMS        *int      `json:"latency_ms"`
	TTFTMS           *int      `json:"ttft_ms"`
	APIKeyHash       *string   `json:"api_key_hash"`
	PolicyViolations *string   `json:"policy_violations"` // JSON array of violations, or null
	CreatedAt        time.Time `json:"created_at"`
}

type Session struct {
	ID          string     `json:"id"`
	Agent       *string    `json:"agent"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	TraceCount  int        `json:"trace_count"`
	TotalTokens int        `json:"total_tokens"`
	TotalCost   float64    `json:"total_cost"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type GraphNode struct {
	ID              string    `json:"id"`
	TraceID         string    `json:"trace_id"`
	SessionID       string    `json:"session_id"`
	Type            string    `json:"type"`
	Agent           *string   `json:"agent"`
	Step            *string   `json:"step"`
	Model           *string   `json:"model"`
	Tokens          *int      `json:"tokens"`
	Cost            *float64  `json:"cost"`
	LatencyMS       *int      `json:"latency_ms"`
	ContextSnapshot *string   `json:"context_snapshot"`
	Confidence      float64   `json:"confidence"`
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
