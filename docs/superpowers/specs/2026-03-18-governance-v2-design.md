# Corvade v2 — Governance & Policy Enforcement

**Date:** 2026-03-18
**Status:** Design approved, pending implementation plan

## Overview

Adds a policy engine to the Corvade proxy that evaluates configurable rules against every intercepted request before forwarding to the LLM. Rules can observe (log violations without blocking) or enforce (block the request and return an error). Violations are captured in traces, shown in the dashboard, and sent to webhooks.

**Goal:** Give teams control over what their agents can do — budget limits, model restrictions, rate limits, PII protection, and cost ceilings — without modifying agent code.

## Architecture

### Request Pipeline (v2)

```
Agent Request
     ↓
  Parse request (model, tokens, headers)
     ↓
  Extract Corvade headers (agent, session, step)
     ↓
  ── NEW: Policy Evaluation ──
  │  Load rules from ~/.corvade/policies.yaml
  │  Evaluate each rule in order:
  │    PASS → continue
  │    OBSERVE violation → log, continue
  │    ENFORCE violation → return error, log, webhook alert
  ├──────────────────────────
     ↓
  Forward to LLM (if not blocked)
     ↓
  Stream response back to agent
     ↓
  Capture trace (with policy_violations attached)
```

Policy evaluation is synchronous in the request path. All rules must be fast (sub-millisecond for numeric checks, low-millisecond for PII regex).

### New Packages

| Package | Responsibility |
|---------|---------------|
| `internal/policy/engine.go` | Load policies, watch for file changes, evaluate rules, return violations |
| `internal/policy/rules/token_limit.go` | Token limit rule implementation |
| `internal/policy/rules/model_allowlist.go` | Model allowlist/blocklist rule |
| `internal/policy/rules/rate_limit.go` | Sliding window rate limiter |
| `internal/policy/rules/pii_redaction.go` | Regex-based PII scanning and redaction |
| `internal/policy/rules/cost_ceiling.go` | Cumulative cost tracking with budget limits |
| `internal/policy/webhook.go` | Async webhook delivery for violation alerts |

## Section 1: Policy File Format

### Location

`~/.corvade/policies.yaml` — separate from the main `config.yaml`.

### Schema

```yaml
# Optional webhook for violation alerts
webhook:
  url: https://hooks.slack.com/services/T00/B00/xxx
  events: [enforce]    # observe | enforce | all (default: enforce only)

# Policy rules — evaluated in order
rules:
  - name: token-limit           # unique name for this rule
    type: token_limit           # rule type
    mode: observe               # observe | enforce
    config:
      max_tokens: 10000

  - name: prod-models
    type: model_allowlist
    mode: enforce
    config:
      allowed: [gpt-4o-mini, claude-3-5-haiku-20241022]

  - name: global-rate
    type: rate_limit
    mode: enforce
    config:
      max_requests: 100
      window: 1m
      per: global

  - name: pii-scan
    type: pii_redaction
    mode: enforce
    config:
      patterns: [email, phone, ssn, credit_card]
      action: redact

  - name: agent-budget
    type: cost_ceiling
    mode: enforce
    config:
      max_cost: 5.00
      window: 24h
      per: agent
```

### Hot Reload

- Policy file watched via `os.Stat` polling every 2 seconds (mtime check)
- On change: parse → validate → atomically swap rule set (mutex-protected)
- If new file is invalid: log error, keep previous rules active
- If file doesn't exist: zero policies (v1 transparent proxy behavior)
- No restart required

### Why Polling Over fsnotify

Simpler, no cgo dependency, works on all platforms including WSL. 2-second delay is acceptable — policy changes aren't latency-sensitive.

## Section 2: Rule Types

### Token Limit (`token_limit`)

Estimates total tokens in the request. Uses word count × 1.3 as approximation. If request includes `max_tokens`, adds that (worst case: prompt + max completion).

**Token estimation across providers:** `estimateTokens(reqBody)` must handle both:
- **OpenAI:** Concatenate all `messages[].content` values (string or array of content parts)
- **Anthropic:** Concatenate all `messages[].content` values + the top-level `system` string if present

Both use the same word-count × 1.3 approximation after concatenation.

```yaml
config:
  max_tokens: 10000
```

Violation message: `"Estimated 12,450 tokens exceeds limit of 10,000"`

### Model Allowlist (`model_allowlist`)

Checks request `model` field against an allow or block list. Supports `allowed` OR `blocked` (not both).

```yaml
config:
  allowed: [gpt-4o-mini, gpt-4o]
  # OR
  blocked: [gpt-4-turbo, claude-3-opus-20240229]
```

Validation: error if both `allowed` and `blocked` are set.

Violation message: `"Model gpt-4o is not in the allowed list [gpt-4o-mini]"`

### Rate Limit (`rate_limit`)

Sliding window counter tracking request count.

```yaml
config:
  max_requests: 100
  window: 1m        # 1m, 5m, 1h, 24h
  per: global       # global | agent | model
```

**Implementation:**
- In-memory map: `{scope_key: [timestamps]}`
- Prune expired entries on each check
- Persist totals to SQLite every 10 seconds for crash recovery
- Hot path is memory-only (no DB queries during evaluation)

Scope keys:
- `global` → single counter
- `agent` → keyed by `X-Corvade-Agent` header value (or "unknown")
- `model` → keyed by request model field

Violation message: `"Rate limit exceeded: 101/100 requests in 1m (global)"`

### PII Redaction (`pii_redaction`)

Regex-based scanning of request body content (messages array text content).

```yaml
config:
  patterns: [email, phone, ssn, credit_card]
  action: redact    # block | redact (default: block)
```

**Built-in patterns:**

| Pattern | Regex | Example Match |
|---------|-------|--------------|
| email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | user@example.com |
| phone | `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b` | 555-123-4567 |
| ssn | `\b\d{3}-\d{2}-\d{4}\b` | 123-45-6789 |
| credit_card | `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b` | 4111-1111-1111-1111 |

**Behavior:**
- **observe mode:** Log violation with match count, forward request unchanged
- **enforce + block:** Return error to agent, do not forward
- **enforce + redact:** Replace all matches with `[REDACTED]` in the request body, then forward the modified request. **The redacted body is stored in the trace** (not the original) — if PII was redacted before sending to the LLM, it should also be redacted in Corvade's database. The violation message records what was redacted (pattern counts) without including the actual PII values.

Regexes are compiled once at policy load time, not per-request.

Violation message: `"PII detected: 2 email addresses, 1 phone number found in request"`

### Cost Ceiling (`cost_ceiling`)

Tracks cumulative spend per scope within a time window.

```yaml
config:
  max_cost: 5.00
  window: 24h       # 1h, 24h, 7d, 30d
  per: agent        # agent | session | global
```

**Implementation:**
- Query SQLite: `SELECT COALESCE(SUM(cost), 0) FROM traces WHERE agent = ? AND created_at >= ?`
- Cache result in memory for 5 seconds (avoid DB hit on every request)
- Estimate current request cost: prompt tokens × model input price (from cost calculator)
- Check: cached_spend + estimated_cost > max_cost?

Violation message: `"Cost ceiling exceeded: $4.92 spent + $0.15 estimated = $5.07 > $5.00 limit (agent: research-bot, window: 24h)"`

**Known limitation:** The 5-second cache creates a TOCTOU race under concurrent load — multiple requests can read the same cached spend, all pass, and collectively exceed the ceiling before the cache refreshes. For Corvade's primary audience (solo devs and small teams), this is acceptable. The ceiling is a guardrail, not a hard financial limit. Acknowledged as a known limitation, not a bug.

## Section 3: Policy Engine

### Interface

```go
type Violation struct {
    Rule    string `json:"rule"`     // rule name
    Type    string `json:"type"`     // rule type
    Mode    string `json:"mode"`     // observe | enforce
    Message string `json:"message"`  // human-readable violation description
}

type EvalContext struct {
    Provider      string
    Model         string
    Agent         string
    Session       string
    Step          string   // from X-Corvade-Step header, for consistency
    RequestBody   []byte
    TokenEstimate int
    EstimatedCost float64  // calc.Calculate(model, tokenEstimate, 0) — 0 completion tokens is intentional pre-request estimate
}

type EvalResult struct {
    Violations []Violation
    Blocked    bool        // true if any enforce-mode rule was violated
    Modified   []byte      // non-nil if PII redaction modified the request body
}

type Engine struct { ... }

func NewEngine(policyPath string, store *capture.Store, calc *cost.Calculator) *Engine
func (e *Engine) Evaluate(ctx EvalContext) EvalResult
func (e *Engine) Start()  // starts file watcher goroutine
func (e *Engine) Stop()   // stops file watcher
func (e *Engine) Rules() []RuleSummary  // for corvade policies command
```

### Rule Interface

Each rule type implements:

```go
type Rule interface {
    Name() string
    Type() string
    Mode() string  // "observe" | "enforce"
    Evaluate(ctx EvalContext) *Violation  // nil = pass, non-nil = violation
}
```

### Evaluation Flow

```go
func (e *Engine) Evaluate(ctx EvalContext) EvalResult {
    result := EvalResult{}
    for _, rule := range e.rules {
        violation := rule.Evaluate(ctx)
        if violation == nil {
            continue
        }
        result.Violations = append(result.Violations, *violation)
        if violation.Mode == "enforce" {
            result.Blocked = true
            break  // stop evaluating after first enforced violation
        }
    }
    if result.Blocked {
        go e.sendWebhook(result.Violations)
    }
    return result
}
```

Note: stops on first enforce violation (no point checking more rules if the request is blocked). Observe violations accumulate — all are logged.

**Webhook call:** The engine calls `sendWebhook` whenever any violations occur (not just when blocked). The `sendWebhook` function filters by the configured `events` list:
- `events: [enforce]` → only send for enforce-mode violations
- `events: [observe]` → only send for observe-mode violations
- `events: [all]` or `events: [observe, enforce]` → send for any violation

Updated evaluation flow:
```go
if len(result.Violations) > 0 {
    go e.sendWebhook(result.Violations)  // sendWebhook filters by events config
}
```

## Section 4: Proxy Integration

### Where It Plugs In

In `internal/proxy/server.go`, inside `handleProxy`, after parsing the request and extracting headers, before forwarding:

```go
// After: reqInfo, _ := providers.OpenAIParseRequest(reqBody)
// Before: upstreamReq, err := http.NewRequestWithContext(...)

if s.policyEngine != nil {
    evalCtx := policy.EvalContext{
        Provider:      provider,
        Model:         model,
        Agent:         corvadeAgent,
        Session:       corvadeSession,
        RequestBody:   reqBody,
        TokenEstimate: estimateTokens(reqBody),
        EstimatedCost: s.calc.Calculate(model, estimateTokens(reqBody), 0),
    }
    result := s.policyEngine.Evaluate(evalCtx)

    if result.Blocked {
        // Return provider-formatted error to agent
        errJSON := writeProviderError(w, provider, result.Violations)
        // Capture the blocked request as a trace for audit trail
        // Reuse captureTrace with: response = error JSON, status_code = 499,
        // latencyMS = time.Since(start), ttft_ms = nil, violations = result.Violations
        go s.captureTrace(provider, model, string(reqBody), errJSON, 499,
            nil, int(time.Since(start).Milliseconds()), apiKeyHash,
            corvadeAgent, corvadeSession, corvadeStep, result.Violations)
        return
    }

    if result.Modified != nil {
        reqBody = result.Modified  // PII-redacted body
    }

    // Pass violations to the captureTrace call at the bottom of handleProxy
    // (captureTrace signature gains a new `violations []policy.Violation` parameter)
    policyViolations = result.Violations
}

// ... existing forwarding code ...

// At the bottom of handleProxy, the existing captureTrace call gains the violations param:
go s.captureTrace(provider, model, ..., policyViolations)
```

### Blocked Response Format

**OpenAI format:**
```json
{
  "error": {
    "message": "Request blocked by Corvade policy: token_limit exceeded (12,450 > 10,000)",
    "type": "policy_violation",
    "code": "corvade_policy_blocked"
  }
}
```

**Anthropic format:**
```json
{
  "type": "error",
  "error": {
    "type": "policy_violation",
    "message": "Request blocked by Corvade policy: token_limit exceeded (12,450 > 10,000)"
  }
}
```

Response headers: `X-Corvade-Policy-Violation: true`, HTTP status `400`.

### Trace Schema Change

```sql
ALTER TABLE traces ADD COLUMN policy_violations TEXT;

CREATE TABLE IF NOT EXISTS rate_limit_state (
    scope_key   TEXT PRIMARY KEY,   -- e.g. "global", "agent:research-bot", "model:gpt-4o"
    window_start TEXT NOT NULL,      -- ISO 8601 timestamp
    count       INTEGER DEFAULT 0,
    updated_at  TEXT NOT NULL
);
```

Stores a JSON array of violations (or NULL if none). Example:
```json
[
  {"rule": "token-limit", "type": "token_limit", "mode": "observe", "message": "Estimated 12,450 tokens exceeds limit of 10,000"}
]
```

Blocked traces use `status_code = 499` (Corvade-specific, similar to nginx's 499).

## Section 5: Webhook Alerting

### Configuration

```yaml
webhook:
  url: https://hooks.slack.com/services/T00/B00/xxx
  events: [enforce]    # observe | enforce | all
```

### Payload

```json
{
  "event": "policy_violation",
  "timestamp": "2026-03-18T15:30:00Z",
  "rule": {
    "name": "token-limit",
    "type": "token_limit",
    "mode": "enforce"
  },
  "violation": {
    "message": "Estimated 12,450 tokens exceeds limit of 10,000",
    "blocked": true
  },
  "request": {
    "model": "gpt-4o",
    "agent": "research-bot",
    "session": "sess-001",
    "provider": "openai"
  }
}
```

### Delivery

- Fire-and-forget goroutine (never blocks the proxy)
- 5-second HTTP timeout
- No retries in v2 (violations are always persisted in the database regardless)
- Failed deliveries logged to stderr

## Section 6: Dashboard Changes

### Traces Tab

- **Shield badge:** Small icon before the status code on rows with violations
  - Violet shield: observe-mode violations only
  - Red shield: enforce-mode violation (request was blocked)
- **Blocked rows:** `status_code: 499`, get `.row-error` diagonal stripe treatment
- **"Violations only" filter:** Toggle button in the filter bar that shows only traces with policy violations

### Detail Inspector

- **New "Policy" tab** alongside Request and Response
- Lists each violation: rule name, type, mode badge (violet=observe, red=enforce), message
- If blocked: banner at top — "This request was blocked by policy enforcement"

### Stats Tab

- **New "Policy Violations" section** below existing breakdowns
- Violations by rule: bar chart (same CSS bar style as By Model)
- Violations by mode: observe count vs enforce count

### API Changes

- `GET /api/traces` response: each trace object now includes `policy_violations` field (JSON array or null)
- `GET /api/stats` response: adds `violations_by_rule: {}` and `violations_by_mode: {observe: N, enforce: N}`

No new API endpoints. Policy data is attached to existing trace/stats data.

**WebSocket event update:** The `trace:new` event payload adds a `policy_violations` field (array or null). For blocked requests, a `trace:blocked` event is emitted instead of `trace:new`:
```json
{ "event": "trace:blocked", "data": { "id": "...", "model": "gpt-4o", "agent": "research-bot", "policy_violations": [...] } }
```
The `corvade tail` command listens for both `trace:new` and `trace:blocked` events and renders the appropriate indicator.

## Section 7: CLI Changes

### `corvade tail` — Violation Indicators

```
12:04:01 gpt-4o      │ $0.02 │ 1.3s │ ✓ summarize results
12:04:03 gpt-4o      │    -  │    - │ ⛨ BLOCKED: token_limit exceeded
12:04:05 gpt-4o      │ $0.01 │ 0.8s │ ⚠ observe: pii_redaction (2 matches)
```

- Blocked: `⛨ BLOCKED: {rule_type} {short_message}`
- Observe: `⚠ observe: {rule_type} ({detail})`
- Clean: unchanged (existing `✓` format)

### New Command: `corvade policies`

```bash
corvade policies           # List policies from ~/.corvade/policies.yaml (reads file directly, no running proxy needed)
corvade policies validate  # Validate policies.yaml syntax and rules
```

**`corvade policies` output:**
```
  Rule              Type             Mode      Status
  token-limit       token_limit      observe   ✓ valid
  prod-models       model_allowlist  enforce   ✓ valid
  global-rate       rate_limit       enforce   ✓ valid
  pii-scan          pii_redaction    enforce   ✓ valid
  agent-budget      cost_ceiling     enforce   ✓ valid

  5 rules loaded from ~/.corvade/policies.yaml
  Webhook: configured (https://hooks.slack.com/...)
```

**`corvade policies validate` output (success):**
```
  ✓ policies.yaml is valid (5 rules)
```

**`corvade policies validate` output (error):**
```
  ✗ Rule "bad-rule" has invalid type "unknown_type"
  ✗ Rule "double-list" cannot have both allowed and blocked
  2 errors found
```

Exit code 0 on success, 1 on validation errors. Useful for CI pipelines.

## Section 8: Files Changed

### New Files

| File | Responsibility |
|------|---------------|
| `internal/policy/engine.go` | Policy engine: load, watch, evaluate, manage rules |
| `internal/policy/engine_test.go` | Engine tests |
| `internal/policy/webhook.go` | Async webhook delivery |
| `internal/policy/webhook_test.go` | Webhook tests |
| `internal/policy/rules/token_limit.go` | Token limit rule |
| `internal/policy/rules/model_allowlist.go` | Model allowlist/blocklist rule |
| `internal/policy/rules/rate_limit.go` | Rate limit rule with in-memory counters |
| `internal/policy/rules/pii_redaction.go` | PII regex scanning and redaction |
| `internal/policy/rules/cost_ceiling.go` | Cost ceiling with cached DB queries |
| `internal/policy/rules/rules_test.go` | Tests for all rule types |
| `internal/cli/policies.go` | `corvade policies` command |
| `internal/cli/policies_test.go` | Policies command tests |
| `dashboard/src/components/PolicyBadge.tsx` | Shield badge component |
| `dashboard/src/components/PolicyTab.tsx` | Policy violations tab in detail inspector |

### Modified Files

| File | Changes |
|------|---------|
| `internal/proxy/server.go` | Add policy engine field, evaluate before forwarding, handle blocked requests |
| `internal/capture/store.go` | Add `policy_violations` column, migration, update InsertTrace signature (add violations param), add `HasViolations *bool` to TraceFilter with `WHERE policy_violations IS NOT NULL` clause, add `ViolationsByRule map[string]int` and `ViolationsByMode struct{Observe, Enforce int}` to Stats struct, add violation aggregation query to GetStats |
| `internal/capture/models.go` | Add `PolicyViolations *string` field (JSON) to Trace struct |
| `internal/api/handlers/export.go` | Add violation stats to GetStats response |
| `internal/cli/start.go` | Initialize policy engine, pass to proxy |
| `internal/cli/tail.go` | Show violation indicators |
| `cmd/corvade/main.go` | Register policies command |
| `dashboard/src/components/Timeline.tsx` | Add shield badges, violations filter |
| `dashboard/src/components/DetailInspector.tsx` | Add Policy tab |
| `dashboard/src/app/stats/page.tsx` | Add violations section |

## Non-Goals (v2)

- Response scanning (checking LLM output for PII/policy violations)
- Custom regex patterns for PII (built-in only in v2)
- Policy inheritance or composition (each rule is independent)
- User-defined rule types (plugin system)
- Audit trail PDF/Markdown export (deferred — the data is in the DB, export is a presentation concern)
- Per-rule webhook URLs (single global webhook in v2)
