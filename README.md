# Corvade

**See what your agents see.**

Corvade is a local AI agent control plane that sits between your agents and LLM APIs as a transparent proxy, capturing every call and visualizing execution as a directed topology graph — not a flat list of traces.

![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go) ![License MIT](https://img.shields.io/badge/license-MIT-green?style=flat-square) ![Status v0.1.0](https://img.shields.io/badge/version-0.1.0-violet?style=flat-square)

---

## What is Corvade?

Most agent observability tools show you what happened. Corvade shows you why — capturing the full causal graph of agent execution as a DAG, inferring relationships between LLM calls without requiring SDK instrumentation, and making agent behavior reviewable by anyone on your team, not just engineers.

Change one environment variable. Point your agent at `http://localhost:4400/v1`. Get topology.

**Target audience:** Solo developers and small engineering teams (2–20 people) building AI agent systems.

---

## Features

- **Zero-config proxy** — one env var change intercepts all agent traffic. No SDK required, no code changes.
- **Topology graph visualization** — interactive DAG (React Flow + Dagre layout) showing causal relationships between LLM calls, not just a flat timeline.
- **Three-strategy topology inference** — tool call ID tracking (conf. ~0.9), conversation fingerprinting (conf. ~0.7), timing gaps (conf. ~0.5). Applied in priority order.
- **Session diff** — select any two sessions and see exactly where their execution paths diverged.
- **Narrative view** — template-based plain English rendering of agent sessions. Readable by PMs, QA, and compliance, not just engineers. Optional LLM-enhancement cached after first generation.
- **Real-time dashboard** — live WebSocket updates push new traces to the dashboard instantly without polling.
- **SSE streaming passthrough** — streaming responses flow through in real-time. The proxy assembles the full response in the background without buffering it for the agent.
- **Cost tracking** — per-trace USD cost calculated from token counts. Built-in pricing for 12+ OpenAI and Anthropic models. Per-model overrides via config.
- **Multi-provider support** — any OpenAI-compatible API on `/v1/*`, native Anthropic adapter on `/anthropic/*`. Single proxy handles both.
- **Demo mode** — `corvade start --demo` loads embedded fixture traces so you can explore the dashboard immediately without running an agent.
- **Living topology canvas** — animated graph background on the dashboard home screen.
- **Command palette** — `Cmd+K` search across traces, sessions, and navigation actions.
- **Keyboard navigation** — full keyboard control of the trace table and detail inspector. Tab switching via `1`–`4`.
- **Agent avatars** — consistent visual identity per agent name across all views.
- **Stats tab** — cost and usage analytics by model and agent, with time range filtering. CSS bars and SVG sparklines, no charting library.
- **SQLite storage** — local, zero-dependency persistence in WAL mode. Configurable retention policy.
- **CLI tools** — `tail`, `doctor`, `sessions`, `inspect` for terminal-native workflows.

---

## Quick Start

### Install

```bash
# Go developers
go install github.com/corvade/corvade@latest

# macOS (coming soon)
brew install corvade

# Any platform
curl -fsSL https://corvade.dev/install.sh | sh
```

### Start with demo data

```bash
corvade start --demo
```

The dashboard opens at `http://localhost:4401`. Demo traces are loaded so you can explore the UI immediately.

### Point your agent at the proxy

```bash
# OpenAI and OpenAI-compatible APIs
export OPENAI_BASE_URL=http://localhost:4400/v1

# Anthropic
export ANTHROPIC_BASE_URL=http://localhost:4400/anthropic
```

That is all that is required. Your agent's API key passes through transparently — Corvade stores only a SHA-256 hash and never the raw key.

### Run your agent

Traffic appears in the dashboard in real time. The terminal shows the startup banner:

```
  ▗▖  Corvade v0.1.0
  ▝▘  The AI agent control plane

  Proxy:      http://localhost:4400
  Dashboard:  http://localhost:4401
  Storage:    ~/.corvade/corvade.db (0 B)

  Point your agents here:
    OPENAI_BASE_URL=http://localhost:4400/v1
    ANTHROPIC_BASE_URL=http://localhost:4400/anthropic

  Watching for traffic...
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Your Agent                           │
└─────────────────────────┬───────────────────────────────────┘
                          │  OPENAI_BASE_URL=http://localhost:4400/v1
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Corvade Proxy  (port 4400)                     │
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────┐  │
│  │ Path Router  │   │ Provider     │   │ SSE Passthrough│  │
│  │ /v1/*        │──▶│ Adapter      │──▶│ (streaming)    │  │
│  │ /anthropic/* │   │ OpenAI /     │   └────────────────┘  │
│  └──────────────┘   │ Anthropic    │                        │
│                     └──────┬───────┘                        │
│                            │ async, best-effort             │
│                            ▼                                │
│                   ┌────────────────┐                        │
│                   │   SQLite DB    │  ~/.corvade/corvade.db  │
│                   │   WAL mode     │                        │
│                   └────────┬───────┘                        │
└────────────────────────────┼────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│              API Server  (port 4401)                        │
│                                                             │
│  ┌──────────────────┐   ┌────────────────────────────────┐  │
│  │  REST API        │   │  WebSocket Hub                 │  │
│  │  /api/traces     │   │  ws://localhost:4401/ws        │  │
│  │  /api/sessions   │   │                                │  │
│  │  /api/stats      │   │  Events: trace:new             │  │
│  └──────────────────┘   │           session:new          │  │
│                         │           session:update       │  │
│  ┌──────────────────┐   └────────────────────────────────┘  │
│  │  Topology Engine │                                        │
│  │  Tool call IDs   │                                        │
│  │  Conv. fingerprint│                                       │
│  │  Timing gaps     │                                        │
│  └──────────────────┘                                        │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│           Next.js Dashboard  (static, embedded)             │
│                                                             │
│   Traces │ Sessions │ Diff │ Stats                          │
│                                                             │
│   Timeline · Topology Graph · Narrative View                │
│   Session Diff · Stats · Command Palette                    │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
                   Real LLM APIs
             (OpenAI, Anthropic, Groq, ...)
```

### Layers

| Layer | Responsibility |
|-------|---------------|
| Proxy (4400) | Intercepts agent requests, forwards to real LLM, captures traces asynchronously. Capture failures never interrupt the agent. |
| SQLite | Local WAL-mode database. Stores raw traces, derived graph nodes/edges, and session summaries. Configurable 30-day retention. |
| Topology Engine | Infers causal graph from captured traces using three strategies applied in priority order. Runs on each session graph request. |
| API Server (4401) | Serves the REST API, WebSocket hub, and embedded Next.js static files from a single Go binary. |
| Dashboard | Next.js app, statically exported and embedded in the Go binary via `go:embed`. No separate Node process in production. |

---

## Dashboard

The dashboard runs at `http://localhost:4401` and is served as a static Next.js export embedded in the Go binary. No separate process is needed.

### Tabs

| Tab | URL | What it shows |
|-----|-----|----------------|
| Traces | `/` | Reverse-chronological feed of every intercepted LLM call with real-time WebSocket updates |
| Sessions | `/sessions` | Card grid of all agent sessions with trace counts, costs, and status badges |
| Diff | `/diff` | Side-by-side session comparison with divergence highlighting |
| Stats | `/stats` | Cost and usage analytics by model and agent with time range filtering |

### Traces Tab

The trace table shows every captured LLM call: timestamp (relative, e.g. "2s ago"), model, agent label, token count, cost, latency, and status code. Rows are color-coded green (2xx), yellow (429), red (4xx/5xx).

Click any row to expand the **Detail Inspector** inline below it. The inspector shows:
- Request and response JSON with line numbers
- Token counts per tab: "Request (25 tok)" / "Response (15 tok)"
- Full prompt and completion with syntax highlighting

A live stats bar above the table shows aggregate counts for the current filter — traces, tokens, cost, agents, models — updated on each WebSocket `trace:new` event.

### Sessions Tab

Each session card shows the agent name, trace count, total tokens, total cost, status badge (active/completed/error), time range, and model chips. Clicking a card opens the session detail page.

The **session detail page** uses a two-panel layout:
- Left (60%): Interactive topology graph (React Flow + Dagre)
- Right (40%): Segmented control toggling between Narrative View and Trace List

### Diff Tab

Select two sessions via combobox inputs. Typing filters by agent name or ID prefix. The diff results show:
- Summary bar: "12 matched · 3 diverged · 2 left-only · 1 right-only"
- Matched pairs side by side with a match percentage bar
- Divergence points highlighted with a pulsing indicator
- Left-only and right-only nodes in distinct sections

### Stats Tab

Filterable by time range: Last hour, Last 24h, Last 7d, Last 30d, All time. Shows total traces, total cost, total tokens, and average cost per request. Breakdowns by model and by agent rendered as CSS bars. Cost over time shown as an SVG sparkline.

### Keyboard Shortcuts

| Key | Scope | Action |
|-----|-------|--------|
| `1` | Global | Switch to Traces tab |
| `2` | Global | Switch to Sessions tab |
| `3` | Global | Switch to Diff tab |
| `4` | Global | Switch to Stats tab |
| `?` | Global | Show keyboard shortcut help overlay |
| `Cmd+K` | Global | Open command palette |
| `j` / `↓` | Traces | Move selection down |
| `k` / `↑` | Traces | Move selection up |
| `Enter` / `Space` | Traces | Toggle detail inspector |
| `Escape` | Traces | Close detail inspector |
| `/` | Traces | Focus search input |

All shortcuts are no-ops when an `input`, `textarea`, or `select` is focused.

---

## CLI Reference

### `corvade start`

Starts the proxy server, API server, and WebSocket hub.

```bash
corvade start [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--demo` | false | Load embedded demo traces on startup |
| `--headless` | false | Start proxy only — no API server or dashboard |
| `--port` | 4400 | Proxy listen port |
| `--dashboard-port` | 4401 | API server and WebSocket port |

```bash
# Standard start
corvade start

# Demo mode — explore the dashboard without running an agent
corvade start --demo

# Proxy only, for headless environments
corvade start --headless

# Custom ports if defaults are taken
corvade start --port 5400 --dashboard-port 5401
```

On first run, Corvade creates `~/.corvade/config.yaml` with defaults and reports the path.

---

### `corvade tail`

Live-streams trace events from a running Corvade instance to stdout. Connects via WebSocket to the API server.

```bash
corvade tail [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--agent` | (all) | Filter output to a specific agent name |
| `--model` | (all) | Filter output to a specific model name |
| `--port` | 4401 | API server port to connect to |

```bash
# All traffic
corvade tail

# Filter by agent
corvade tail --agent research-bot

# Filter by model
corvade tail --model gpt-4o

# Combined
corvade tail --agent research-bot --model claude-3-5-sonnet-20241022
```

**Example output:**

```
  Connecting to ws://localhost:4401/ws ...
  Connected. Watching for traces...

12:04:01 gpt-4o         │  $0.02 │  1.3s │ ✓ openai
12:04:03 gpt-4o         │  $0.01 │  0.8s │ ✓ openai
12:04:04 claude-sonnet  │  $0.03 │  1.1s │ ✓ anthropic
```

Press `Ctrl+C` to disconnect cleanly.

---

### `corvade doctor`

Runs pre-flight diagnostics. Checks port availability, SQLite writability, and available disk space.

```bash
corvade doctor
```

**Example output:**

```
  ✓ Port 4400 available
  ✓ Port 4401 available
  ✓ SQLite writable at /home/user/.corvade/corvade.db
  ✓ 14.2 GB disk space available
```

Run this first if `corvade start` fails or if the dashboard is not receiving traffic.

---

### `corvade sessions`

Lists recent agent sessions from the local SQLite database. Does not require Corvade to be running.

```bash
corvade sessions [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 20 | Maximum number of sessions to display |

```bash
corvade sessions
corvade sessions --limit 50
```

**Example output:**

```
ID          AGENT          TRACES  COST   STATUS     STARTED
──          ─────          ──────  ────   ──────     ───────
01JE1B4...  research-bot   12      $0.18  completed  2026-03-18T12:04:01Z
01JE1A9...  -              3       $0.04  active     2026-03-18T11:51:22Z
```

---

### `corvade inspect <id>`

Prints the full JSON record for a trace or session ID. Tries the ID as a trace first, then as a session. Does not require Corvade to be running.

```bash
corvade inspect <id>
```

```bash
# Inspect a trace
corvade inspect 01JE1B4KZXYZ123

# Inspect a session
corvade inspect 01JE1A9MWXYZ456
```

Output is formatted JSON sent to stdout. Suitable for piping to `jq`.

---

### `corvade version`

Prints the current version.

```bash
corvade version
# corvade v0.1.0
```

---

## API Reference

The API server runs on port 4401. All endpoints return `application/json`. The WebSocket endpoint is at `ws://localhost:4401/ws`.

### GET /api/traces

List captured traces with optional filtering.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `agent` | string | Filter by agent name |
| `model` | string | Filter by model name |
| `status` | integer | Filter by HTTP status code |
| `from` | ISO 8601 | Start of time range |
| `to` | ISO 8601 | End of time range |
| `search` | string | Full-text search across prompts and completions |
| `limit` | integer | Maximum results (default 50) |
| `offset` | integer | Pagination offset (default 0) |

<details>
<summary>Response schema</summary>

```json
[
  {
    "id": "01JE1B4KZXYZ123",
    "session_id": "01JE1A9MWXYZ456",
    "agent": "research-bot",
    "step": "summarize",
    "provider": "openai",
    "model": "gpt-4o",
    "request": "{...}",
    "response": "{...}",
    "status_code": 200,
    "tokens_prompt": 892,
    "tokens_completion": 341,
    "tokens_cached": 0,
    "cost": 0.005635,
    "latency_ms": 1312,
    "ttft_ms": 210,
    "api_key_hash": "a3f2c1...",
    "created_at": "2026-03-18T12:04:01Z"
  }
]
```

</details>

---

### GET /api/traces/:id

Single trace by ID.

```
GET /api/traces/01JE1B4KZXYZ123
```

Returns a single trace object. Returns 404 if not found.

---

### GET /api/sessions

List sessions with optional filtering.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `agent` | string | Filter by agent name |
| `from` | ISO 8601 | Start of time range |
| `to` | ISO 8601 | End of time range |
| `limit` | integer | Maximum results (default 20) |
| `offset` | integer | Pagination offset |

<details>
<summary>Response schema</summary>

```json
[
  {
    "id": "01JE1A9MWXYZ456",
    "agent": "research-bot",
    "start_time": "2026-03-18T12:04:01Z",
    "end_time": "2026-03-18T12:09:45Z",
    "trace_count": 12,
    "total_tokens": 8431,
    "total_cost": 0.1823,
    "status": "completed",
    "created_at": "2026-03-18T12:04:01Z"
  }
]
```

</details>

---

### GET /api/sessions/:id

Single session by ID with summary stats.

```
GET /api/sessions/01JE1A9MWXYZ456
```

---

### GET /api/sessions/:id/graph

Topology graph for a session — nodes and edges inferred by the topology engine.

```
GET /api/sessions/01JE1A9MWXYZ456/graph
```

<details>
<summary>Response schema</summary>

```json
{
  "nodes": [
    {
      "id": "01JE1B4NODE001",
      "trace_id": "01JE1B4KZXYZ123",
      "session_id": "01JE1A9MWXYZ456",
      "type": "completion",
      "agent": "research-bot",
      "step": "summarize",
      "model": "gpt-4o",
      "tokens": 1233,
      "cost": 0.005635,
      "latency_ms": 1312,
      "context_snapshot": "You are a research assistant...",
      "confidence": 1.0,
      "created_at": "2026-03-18T12:04:01Z"
    }
  ],
  "edges": [
    {
      "id": "01JE1B4EDGE001",
      "session_id": "01JE1A9MWXYZ456",
      "from_node": "01JE1B4NODE001",
      "to_node": "01JE1B4NODE002",
      "type": "triggered",
      "confidence": 0.9,
      "created_at": "2026-03-18T12:04:03Z"
    }
  ]
}
```

Node `type` values: `request`, `tool_call`, `completion`.
Edge `type` values: `triggered`, `timing`.
`confidence` reflects the inference strategy: `~0.9` (tool call tracking), `~0.7` (conversation fingerprinting), `0.5–0.6` (timing gap).

</details>

---

### GET /api/sessions/:id/narrative

Template-based narrative for a session. Returns structured segments rendered from each graph node using deterministic templates.

```
GET /api/sessions/01JE1A9MWXYZ456/narrative
```

<details>
<summary>Response schema</summary>

```json
{
  "segments": [
    {
      "node_id": "01JE1B4NODE001",
      "text": "The research-bot asked gpt-4o to summarize results",
      "node_type": "completion"
    }
  ]
}
```

</details>

---

### POST /api/sessions/:id/narrative/enhance

Triggers LLM-enhanced narrative generation. Sends session data to the user's own LLM via the proxy. The result is cached — calling this endpoint a second time returns the cached version without making another LLM call.

```
POST /api/sessions/01JE1A9MWXYZ456/narrative/enhance
```

<details>
<summary>Response schema</summary>

```json
{
  "segments": [
    {
      "node_id": "01JE1B4NODE001",
      "text": "The research agent began by asking GPT-4o to distill the key findings from a web search into a concise summary.",
      "node_type": "completion"
    }
  ],
  "model_used": "gpt-4o",
  "tokens_used": 892,
  "cost": 0.00415
}
```

</details>

---

### GET /api/sessions/:id1/diff/:id2

Session diff between two sessions. Aligns nodes using step labels, tool call names, and position-based LCS.

```
GET /api/sessions/01JE1A9MWXYZ456/diff/01JE1B4MWXYZ789
```

<details>
<summary>Response schema</summary>

```json
{
  "aligned_nodes": [
    {
      "left": { ...node },
      "right": { ...node },
      "match_score": 0.94
    }
  ],
  "left_only": [ ...nodes ],
  "right_only": [ ...nodes ],
  "divergence_points": [ ...nodes ]
}
```

</details>

---

### GET /api/stats

Dashboard summary statistics with optional time range filtering.

```
GET /api/stats
GET /api/stats?from=2026-03-18T00:00:00Z
GET /api/stats?from=2026-03-11T00:00:00Z&to=2026-03-18T00:00:00Z
```

<details>
<summary>Response schema</summary>

```json
{
  "total_traces": 847,
  "total_sessions": 42,
  "total_cost": 12.43,
  "total_tokens": 245891,
  "by_model": {
    "gpt-4o": { "traces": 412, "cost": 8.21, "tokens": 142000 },
    "claude-3-5-sonnet-20241022": { "traces": 435, "cost": 4.22, "tokens": 103891 }
  },
  "by_agent": {
    "research-bot": { "traces": 623, "cost": 9.14 },
    "summarizer": { "traces": 224, "cost": 3.29 }
  }
}
```

</details>

---

### WebSocket: ws://localhost:4401/ws

Connect to receive real-time events as traces are captured. The dashboard uses this to update the trace table, stats bar, and session list without polling.

**Events:**

| Event | Description |
|-------|-------------|
| `trace:new` | A new trace was captured. `data` contains `id`, `provider`, `model`. |
| `trace:stream` | A streaming chunk was received. `data` contains `id` and `chunk`. |
| `trace:complete` | A streaming response finished. `data` contains `id`, `tokens`, `cost`. |
| `session:new` | A new session was detected. `data` is a full session object. |
| `session:update` | Session summary stats were updated. `data` is a full session object. |

**Message format:**

```json
{ "event": "trace:new", "data": { "id": "01JE1B4KZXYZ123", "provider": "openai", "model": "gpt-4o" } }
```

The WebSocket auto-reconnects with exponential backoff on disconnect.

---

## Configuration

Corvade creates `~/.corvade/config.yaml` on first `corvade start`. Edit it directly or via `$EDITOR`.

```yaml
# ~/.corvade/config.yaml

# Port for the proxy server (agent traffic)
port: 4400

# Port for the API server, WebSocket hub, and dashboard
dashboard_port: 4401

# Number of days to retain captured traces before automatic purge
retention_days: 30

# Timing gap threshold for topology inference (milliseconds).
# Traces within this window are candidates for timing-based edges.
# Increase this for slow local models (Ollama). Decrease for low-latency APIs.
timing_gap_ms: 2000

# Per-model cost overrides. Use this if Corvade's built-in pricing table
# is out of date, or for custom/fine-tuned models.
cost_overrides:
  gpt-4o:
    input_per_1k: 0.0025
    output_per_1k: 0.01
  my-fine-tuned-model:
    input_per_1k: 0.005
    output_per_1k: 0.015
```

### Built-in model pricing

Corvade ships with a pricing table for the following models. This table is updated with each release. Override any entry via `cost_overrides`.

| Model | Input per 1K tokens | Output per 1K tokens |
|-------|--------------------|--------------------|
| `gpt-4o` | $0.0025 | $0.0100 |
| `gpt-4o-mini` | $0.00015 | $0.0006 |
| `gpt-4-turbo` | $0.0100 | $0.0300 |
| `gpt-3.5-turbo` | $0.0005 | $0.0020 |
| `o1` | $0.0150 | $0.0600 |
| `o1-mini` | $0.0030 | $0.0120 |
| `claude-3-5-sonnet-20241022` | $0.0030 | $0.0150 |
| `claude-3-5-haiku-20241022` | $0.0008 | $0.0040 |
| `claude-3-opus-20240229` | $0.0150 | $0.0750 |
| `claude-sonnet-4-6` | $0.0030 | $0.0150 |
| `claude-haiku-4-5-20251001` | $0.0008 | $0.0040 |
| `claude-opus-4-6` | $0.0150 | $0.0750 |

Models not in this table record token counts but show `$0.00` cost. Add them via `cost_overrides`.

---

## Providers

### OpenAI and OpenAI-compatible APIs

Route: `/v1/*`

```bash
export OPENAI_BASE_URL=http://localhost:4400/v1
```

Works with any API that speaks the OpenAI chat completions format:

| Provider | Base URL override |
|----------|------------------|
| OpenAI | `http://localhost:4400/v1` |
| Groq | `http://localhost:4400/v1` (set `OPENAI_BASE_URL`) |
| Together AI | `http://localhost:4400/v1` |
| Mistral | `http://localhost:4400/v1` |
| Ollama | `http://localhost:4400/v1` |
| LM Studio | `http://localhost:4400/v1` |
| Azure OpenAI | `http://localhost:4400/v1` |

Your API key passes through in the `Authorization: Bearer <key>` header unchanged.

### Anthropic

Route: `/anthropic/*`

```bash
export ANTHROPIC_BASE_URL=http://localhost:4400/anthropic
```

Anthropic uses a different API shape (`x-api-key` header, `/v1/messages` endpoint). Corvade has a dedicated adapter that parses the Anthropic request and response formats for accurate token extraction.

---

## SDK

For precise topology without relying on inference, inject three headers on every LLM request. The proxy reads and strips these headers before forwarding upstream — they never reach the LLM provider.

| Header | Value | Description |
|--------|-------|-------------|
| `X-Corvade-Agent` | `string` | Agent name. Groups calls by agent in the topology graph. |
| `X-Corvade-Session` | `string` | Session ID. Associates this call with a session. Use a UUID or any stable identifier per agent run. |
| `X-Corvade-Step` | `string` | Step label. Names this specific call within the session (e.g. `"search"`, `"summarize"`, `"respond"`). |

When these headers are present, topology edges have `confidence: 1.0`. When absent, the topology engine infers relationships — edges are labeled with their confidence score and rendered as dashed lines in the graph at lower confidence values.

### TypeScript

```typescript
import OpenAI from "openai"

const sessionId = crypto.randomUUID()

const client = new OpenAI({
  baseURL: "http://localhost:4400/v1",
  defaultHeaders: {
    "X-Corvade-Agent": "research-bot",
    "X-Corvade-Session": sessionId,
  },
})

// Per-call step annotation
const response = await client.chat.completions.create(
  { model: "gpt-4o", messages: [...] },
  { headers: { "X-Corvade-Step": "summarize" } }
)
```

### Python

```python
from openai import OpenAI
import uuid

session_id = str(uuid.uuid4())

client = OpenAI(
    base_url="http://localhost:4400/v1",
    default_headers={
        "X-Corvade-Agent": "research-bot",
        "X-Corvade-Session": session_id,
    },
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[...],
    extra_headers={"X-Corvade-Step": "summarize"},
)
```

A thin SDK (`@corvade/sdk` for TypeScript, `corvade` for Python) that wraps these headers automatically is planned for v1.1.

---

## Development

### Prerequisites

- Go 1.26+
- Node.js 20+ (dashboard only)

### Build from source

```bash
git clone https://github.com/corvade/corvade
cd corvade

# Build the Go binary (proxy + API server)
go build -o bin/corvade ./cmd/corvade

# Or install to $GOPATH/bin
go install ./cmd/corvade
```

### Development mode

Run the proxy and dashboard separately with hot reload:

```bash
# Terminal 1: Next.js dashboard with hot reload
cd dashboard
npm install
npm run dev
# Dashboard dev server at http://localhost:3000

# Terminal 2: Go proxy and API server
go run ./cmd/corvade start
# Proxy at :4400, API at :4401
```

In development the dashboard connects to the API server at `localhost:4401` via the `NEXT_PUBLIC_API_URL` environment variable.

### Production build

```bash
# Build the dashboard static export
cd dashboard && npm run build

# Build the Go binary (embeds dashboard/out/)
go build -o bin/corvade ./cmd/corvade
```

The binary is fully self-contained — it embeds the dashboard static files via `go:embed` and serves them from the API server.

### Run tests

```bash
go test ./...
go test ./... -cover
```

### Project structure

```
corvade/
├── cmd/corvade/
│   └── main.go                  # CLI entrypoint, command registration
├── internal/
│   ├── proxy/                   # Reverse proxy — port 4400
│   │   ├── server.go            # Core proxy handler, SSE passthrough, trace capture
│   │   ├── middleware.go        # Reserved for future middleware
│   │   └── providers/
│   │       ├── types.go         # RequestInfo, ResponseInfo, ToolCall
│   │       ├── openai.go        # OpenAI request/response parser
│   │       └── anthropic.go     # Anthropic request/response parser
│   ├── api/                     # API server — port 4401
│   │   ├── server.go            # HTTP server, go:embed static files
│   │   ├── routes.go            # Route registration
│   │   ├── hub.go               # WebSocket hub
│   │   └── handlers/
│   │       ├── traces.go        # GET /api/traces, GET /api/traces/:id
│   │       ├── sessions.go      # GET /api/sessions, GET /api/sessions/:id
│   │       ├── topology.go      # GET /api/sessions/:id/graph, /narrative, /enhance
│   │       ├── diff.go          # GET /api/sessions/:id1/diff/:id2
│   │       └── export.go        # GET /api/stats
│   ├── capture/                 # SQLite storage layer
│   │   ├── store.go             # DB open, migrations, query methods
│   │   ├── models.go            # Trace, Session, GraphNode, GraphEdge structs
│   │   └── retention.go        # Automatic data retention purge
│   ├── topology/                # Graph inference engine
│   │   ├── inference.go         # Engine.Infer — three-strategy DAG construction
│   │   ├── fingerprint.go       # Conversation fingerprinting (strategy 2)
│   │   ├── toolcall.go          # Tool call ID extraction (strategy 1)
│   │   └── graph.go             # Node/edge builder helpers
│   ├── cost/
│   │   ├── calculator.go        # Token-based cost calculation
│   │   └── models.go            # Built-in pricing table
│   ├── config/
│   │   └── config.go            # Config struct, Load, DefaultPath
│   └── cli/
│       ├── start.go             # corvade start
│       ├── tail.go              # corvade tail
│       ├── doctor.go            # corvade doctor
│       ├── sessions.go          # corvade sessions
│       ├── inspect.go           # corvade inspect
│       ├── demo.go              # Demo data loader (go:embed testdata/)
│       └── testdata/            # Embedded fixture traces (JSON)
├── dashboard/                   # Next.js app
│   ├── src/
│   │   ├── app/
│   │   │   ├── page.tsx         # Traces tab
│   │   │   ├── sessions/        # Sessions tab
│   │   │   ├── session/         # Session detail (topology + narrative)
│   │   │   ├── diff/            # Diff tab
│   │   │   └── stats/           # Stats tab
│   │   └── components/
│   │       ├── Header.tsx        # Fixed top bar with live stats
│   │       ├── TabNav.tsx        # Tab navigation
│   │       ├── Timeline.tsx      # Trace table with keyboard nav
│   │       ├── DetailInspector.tsx
│   │       ├── TopologyGraph.tsx # React Flow + Dagre
│   │       ├── TopologyCanvas.tsx # Animated background canvas
│   │       ├── NarrativeView.tsx
│   │       ├── SessionDiff.tsx
│   │       ├── SessionCard.tsx
│   │       ├── StatsBar.tsx
│   │       ├── CommandPalette.tsx
│   │       ├── AgentAvatar.tsx
│   │       ├── KeyboardHelp.tsx
│   │       ├── SkeletonRows.tsx
│   │       └── EmptyState.tsx
│   └── package.json
├── go.mod
└── README.md
```

### Key dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI command framework |
| `github.com/mattn/go-sqlite3` | SQLite driver (CGO) |
| `github.com/gorilla/websocket` | WebSocket server and client |
| `github.com/oklog/ulid/v2` | Sortable unique IDs for traces and sessions |
| `gopkg.in/yaml.v3` | Config file parsing |
| Next.js 16 | Dashboard framework |
| React Flow | Topology graph rendering |
| Tailwind CSS | Dashboard styling |

---

## SQLite Schema

```sql
CREATE TABLE traces (
    id                TEXT PRIMARY KEY,  -- ULID (sortable)
    session_id        TEXT,
    agent             TEXT,
    step              TEXT,
    provider          TEXT NOT NULL,     -- "openai" | "anthropic"
    model             TEXT NOT NULL,
    request           TEXT NOT NULL,     -- full JSON request body
    response          TEXT,              -- full JSON response (null during streaming)
    status_code       INTEGER NOT NULL,
    tokens_prompt     INTEGER,
    tokens_completion INTEGER,
    tokens_cached     INTEGER,
    cost              REAL,              -- USD
    latency_ms        INTEGER,           -- total request duration
    ttft_ms           INTEGER,           -- time to first token (streaming only)
    api_key_hash      TEXT,              -- SHA-256 of API key, never raw key
    created_at        TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,
    agent        TEXT,
    start_time   TEXT NOT NULL,
    end_time     TEXT,
    trace_count  INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    total_cost   REAL DEFAULT 0,
    status       TEXT DEFAULT 'active',  -- "active" | "completed" | "error"
    created_at   TEXT NOT NULL
);

CREATE TABLE graph_nodes (
    id               TEXT PRIMARY KEY,
    trace_id         TEXT NOT NULL,
    session_id       TEXT NOT NULL,
    type             TEXT NOT NULL,      -- "request" | "tool_call" | "completion"
    agent            TEXT,
    step             TEXT,
    model            TEXT,
    tokens           INTEGER,
    cost             REAL,
    latency_ms       INTEGER,
    context_snapshot TEXT,               -- first 500 chars of prompt
    confidence       REAL DEFAULT 1.0,   -- 1.0 = SDK-tagged, <1.0 = inferred
    created_at       TEXT NOT NULL
);

CREATE TABLE graph_edges (
    id         TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    from_node  TEXT NOT NULL,
    to_node    TEXT NOT NULL,
    type       TEXT NOT NULL,            -- "triggered" | "timing"
    confidence REAL DEFAULT 1.0,
    created_at TEXT NOT NULL
);
```

---

## Topology Inference

Corvade infers causal relationships between traces without requiring SDK headers. The `topology.Engine` applies three strategies in priority order:

### Strategy 1 — Tool call ID tracking (confidence ~0.9)

If trace B's request contains a `tool_call_id` that matches a tool call ID produced by trace A's response, an edge is created from A to B. This is the highest-confidence strategy because tool call IDs are globally unique identifiers.

### Strategy 2 — Conversation fingerprinting (confidence ~0.7)

If trace B's messages array contains the assistant response from trace A verbatim, B is a continuation of A's conversation. This detects multi-turn conversations and agent loops without any explicit tagging.

### Strategy 3 — Timing gaps (confidence ~0.5–0.6)

For traces not connected by the above strategies, if the gap between consecutive traces is within the `timing_gap_ms` threshold (default 2000ms), a timing-based edge is inferred. Confidence decreases linearly as the gap approaches the threshold.

### Limitations

- Concurrent agents sharing one API key may have misattributed edges. Use `X-Corvade-Session` headers to resolve this.
- Local models (Ollama, LM Studio) have higher latency. Increase `timing_gap_ms` in config.
- Low-confidence edges (`< 0.7`) are rendered as dashed lines in the topology graph.

---

## Resilience

Corvade is transparent. If Corvade breaks, your agents keep working.

| Failure | Behavior |
|---------|----------|
| Upstream unreachable | Proxy returns the upstream error directly to the agent with the original status code |
| Capture fails (SQLite error, disk full) | Response still delivered to agent. Failure logged to stderr. `corvade doctor` checks this. |
| Stream interrupted | Partial response saved with `status_code: 0` and flagged in the dashboard |
| Port conflict | `corvade start` exits with a clear message. Use `--port` and `--dashboard-port` flags. |
| Malformed request | Forwarded as-is to upstream. Error passed through to agent. |
| Dashboard down | Zero effect on the proxy. They run as separate goroutines. |

---

## Roadmap

| Version | Focus | Key Deliverables |
|---------|-------|-----------------|
| **v1** | Proxy + Dashboard + CLI | Go proxy, SQLite capture, topology inference, React Flow graph, session diff, narrative view, CLI tools |
| **v2** | Governance | Policy engine (token limits, blocked models, PII redaction), runtime enforcement at proxy layer, audit trail export, webhook/Slack alerts on violations |
| **v3** | MCP Memory | MCP server for persistent cross-session agent memory, memory timeline in dashboard, memory-influenced decision tracking, memory audit/redact |
| **v4** | Full SDK | Explicit decision point marking, custom metadata, step annotations, alternatives logging |
| **v5** | Team Sync | Push-based sync (local → shared server), Postgres backend, team dashboard with activity feed, cost rollups, shared bookmarks, and an auto-discovered agent registry |

---

## License

MIT — see [LICENSE](LICENSE).
