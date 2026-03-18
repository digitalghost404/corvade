# Corvade v2: Governance & Policy Enforcement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a policy engine to the Corvade proxy that evaluates configurable rules (token limits, model allowlists, rate limits, PII redaction, cost ceilings) against every request before forwarding, with observe/enforce modes, webhook alerting, dashboard integration, and CLI tooling.

**Architecture:** New `internal/policy/` package with a Rule interface, 5 rule implementations, an engine that loads YAML policies with hot-reload, and webhook alerting. Integrates into the existing proxy's `handleProxy` function before the forwarding step. Violations are stored as JSON in a new `policy_violations` column on the traces table. Dashboard and CLI show violation data.

**Tech Stack:** Go, SQLite, YAML (gopkg.in/yaml.v3 — already in go.mod), regex for PII patterns

**Spec:** `docs/superpowers/specs/2026-03-18-governance-v2-design.md`

**TDD:** All tasks follow test-driven development. Write tests first, verify they fail, implement, verify they pass.

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `internal/policy/types.go` | Violation, EvalContext, EvalResult, Rule interface, policy file structs |
| `internal/policy/engine.go` | Load policies, file watcher, evaluate rules, manage lifecycle |
| `internal/policy/engine_test.go` | Engine tests |
| `internal/policy/webhook.go` | Async webhook POST delivery |
| `internal/policy/webhook_test.go` | Webhook tests |
| `internal/policy/tokens.go` | Token estimation from request body (both providers) |
| `internal/policy/tokens_test.go` | Token estimation tests |
| `internal/policy/rules/token_limit.go` | Token limit rule |
| `internal/policy/rules/model_allowlist.go` | Model allowlist/blocklist rule |
| `internal/policy/rules/rate_limit.go` | Sliding window rate limiter |
| `internal/policy/rules/pii_redaction.go` | PII regex scanning and redaction |
| `internal/policy/rules/cost_ceiling.go` | Cost ceiling with cached DB queries |
| `internal/policy/rules/rules_test.go` | Tests for all 5 rule types |
| `internal/cli/policies.go` | `corvade policies` and `corvade policies validate` commands |
| `internal/cli/policies_test.go` | Policies command tests |
| `dashboard/src/components/PolicyBadge.tsx` | Shield icon badge for trace rows |
| `dashboard/src/components/PolicyTab.tsx` | Policy violations panel in detail inspector |

### Modified Files

| File | Changes |
|------|---------|
| `internal/capture/models.go` | Add `PolicyViolations *string` to Trace struct |
| `internal/capture/store.go` | Schema migration (policy_violations column + rate_limit_state table), update InsertTrace, add HasViolations filter, update Stats struct + GetStats |
| `internal/proxy/server.go` | Add policyEngine field, evaluate before forwarding, handle blocked requests, writeProviderError |
| `internal/api/handlers/export.go` | No code changes needed — handler already serializes full Stats struct via writeJSON. Adding fields to Stats in store.go automatically exposes them. Listed for awareness. |
| `internal/cli/start.go` | Initialize policy engine, pass to proxy |
| `internal/cli/tail.go` | Violation indicators for trace:blocked events |
| `cmd/corvade/main.go` | Register policies command |
| `dashboard/src/components/Timeline.tsx` | Shield badges, violations filter |
| `dashboard/src/components/DetailInspector.tsx` | Policy tab |
| `dashboard/src/app/stats/page.tsx` | Violations section |

---

## Task 1: Policy Types & Token Estimation

**Files:**
- Create: `internal/policy/types.go`, `internal/policy/tokens.go`, `internal/policy/tokens_test.go`

- [ ] **Step 1: Write token estimation tests**

Test `EstimateTokens(body []byte) int`:
- OpenAI request with 3 messages: count words in all content fields × 1.3
- Anthropic request with system + messages: include system text
- Request with `max_tokens` field: add to estimate
- Empty/malformed body: return 0
- Content as array (OpenAI vision format): handle gracefully

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/policy/ -v
```

- [ ] **Step 3: Create types.go**

Define all shared types:
```go
package policy

type Violation struct {
    Rule    string `json:"rule"`
    Type    string `json:"type"`
    Mode    string `json:"mode"`
    Message string `json:"message"`
}

type EvalContext struct {
    Provider      string
    Model         string
    Agent         string
    Session       string
    Step          string
    RequestBody   []byte
    TokenEstimate int
    EstimatedCost float64
}

type EvalResult struct {
    Violations []Violation
    Blocked    bool
    Modified   []byte
}

type Rule interface {
    Name() string
    Type() string
    Mode() string
    Evaluate(ctx EvalContext) *Violation
}

type RuleSummary struct {
    Name   string `json:"name"`
    Type   string `json:"type"`
    Mode   string `json:"mode"`
    Valid  bool   `json:"valid"`
    Error  string `json:"error,omitempty"`
}

// Policy file schema
type PolicyFile struct {
    Webhook *WebhookConfig `yaml:"webhook"`
    Rules   []RuleConfig   `yaml:"rules"`
}

type WebhookConfig struct {
    URL    string   `yaml:"url"`
    Events []string `yaml:"events"` // observe, enforce, all
}

type RuleConfig struct {
    Name   string                 `yaml:"name"`
    Type   string                 `yaml:"type"`
    Mode   string                 `yaml:"mode"` // observe | enforce
    Config map[string]interface{} `yaml:"config"`
}
```

- [ ] **Step 4: Implement tokens.go**

`EstimateTokens(body []byte) int`:
- Parse JSON to extract messages content (handle both OpenAI and Anthropic shapes)
- Concatenate all text content + system if present
- Count words (split by whitespace), multiply by 1.3
- If `max_tokens` field exists, add it to the estimate
- Return 0 for unparseable input

- [ ] **Step 5: Run tests**

```bash
go test ./internal/policy/ -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/policy/
git commit -m "feat(v2): add policy types and token estimation"
```

---

## Task 2: Rule Implementations (Token Limit + Model Allowlist)

**Files:**
- Create: `internal/policy/rules/token_limit.go`, `internal/policy/rules/model_allowlist.go`
- Create: `internal/policy/rules/rules_test.go`

- [ ] **Step 1: Write tests for token limit**

Test TokenLimitRule.Evaluate:
- Tokens under limit → nil (pass)
- Tokens over limit → Violation with correct message
- Observe mode: violation has mode="observe"
- Enforce mode: violation has mode="enforce"

- [ ] **Step 2: Write tests for model allowlist**

Test ModelAllowlistRule.Evaluate:
- Model in allowed list → pass
- Model not in allowed list → violation
- Model in blocked list → violation
- Model not in blocked list → pass
- Validation: both allowed and blocked set → error on construction

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/policy/rules/ -v
```

- [ ] **Step 4: Implement token_limit.go**

```go
package rules

type TokenLimitRule struct {
    name      string
    mode      string
    maxTokens int
}

func NewTokenLimitRule(name, mode string, config map[string]interface{}) (*TokenLimitRule, error)
func (r *TokenLimitRule) Name() string
func (r *TokenLimitRule) Type() string  // "token_limit"
func (r *TokenLimitRule) Mode() string
func (r *TokenLimitRule) Evaluate(ctx policy.EvalContext) *policy.Violation
```

- [ ] **Step 5: Implement model_allowlist.go**

```go
type ModelAllowlistRule struct {
    name    string
    mode    string
    allowed []string  // nil if using blocked
    blocked []string  // nil if using allowed
}

func NewModelAllowlistRule(name, mode string, config map[string]interface{}) (*ModelAllowlistRule, error)
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/policy/rules/ -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/policy/rules/
git commit -m "feat(v2): add token_limit and model_allowlist rules"
```

---

## Task 3: Rule Implementations (Rate Limit + PII Redaction + Cost Ceiling)

**Files:**
- Create: `internal/policy/rules/rate_limit.go`, `internal/policy/rules/pii_redaction.go`, `internal/policy/rules/cost_ceiling.go`
- Modify: `internal/policy/rules/rules_test.go`

- [ ] **Step 1: Write tests for rate limit**

Test RateLimitRule.Evaluate:
- First request in window → pass
- N+1th request → violation
- After window expires → counter resets, pass
- Per agent scope: different agents have independent counters
- Per model scope: different models have independent counters

- [ ] **Step 2: Write tests for PII redaction**

Test PIIRedactionRule.Evaluate:
- No PII → pass
- Email in content → violation with count
- Multiple patterns matched → violation lists all
- Action=block → returns violation, Modified is nil
- Action=redact → returns violation, Modified contains redacted body
- Malformed request body → pass (don't crash)

- [ ] **Step 3: Write tests for cost ceiling**

Test CostCeilingRule.Evaluate:
- Under budget → pass
- Over budget → violation
- Different scopes (agent, session, global)
- Note: cost ceiling needs a store reference. For tests, use a temp SQLite store with pre-inserted traces.

- [ ] **Step 4: Implement rate_limit.go**

```go
type RateLimitRule struct {
    name        string
    mode        string
    maxRequests int
    windowDur   time.Duration
    per         string // global | agent | model
    mu          sync.Mutex
    counters    map[string][]time.Time
}
```

Parse window string ("1m", "5m", "1h", "24h") into time.Duration.
On Evaluate: get scope key, prune expired timestamps, check count vs max, add current timestamp.

**Note on persistence:** The spec calls for persisting rate limit state to SQLite every 10 seconds. For v2, defer this — the in-memory counters are sufficient for the target audience (solo devs, small teams). The `rate_limit_state` table schema is created in Task 6 as a placeholder, but no read/write logic is implemented in v2. Persistence can be added in a future version if crash recovery becomes important.

- [ ] **Step 5: Implement pii_redaction.go**

```go
type PIIRedactionRule struct {
    name     string
    mode     string
    patterns []*regexp.Regexp
    patNames []string
    action   string // block | redact
}
```

Compile regexes at construction time. On Evaluate: extract text content from request body, scan with each pattern, count matches. If action=redact and mode=enforce, create Modified body with replacements.

Built-in patterns map: `email`, `phone`, `ssn`, `credit_card` → compiled regex.

- [ ] **Step 6: Implement cost_ceiling.go**

```go
type CostCeilingRule struct {
    name      string
    mode      string
    maxCost   float64
    windowDur time.Duration
    per       string // agent | session | global
    store     *capture.Store
    cache     map[string]cachedSpend
    cacheMu   sync.Mutex
}

type cachedSpend struct {
    amount    float64
    fetchedAt time.Time
}
```

On Evaluate: get scope key, check cache (valid for 5s), query store if cache miss, compare spend + estimated cost vs max.

- [ ] **Step 7: Run all rule tests**

```bash
go test ./internal/policy/rules/ -v
```

- [ ] **Step 8: Commit**

```bash
git add internal/policy/rules/
git commit -m "feat(v2): add rate_limit, pii_redaction, and cost_ceiling rules"
```

---

## Task 4: Policy Engine (Load, Watch, Evaluate)

**Files:**
- Create: `internal/policy/engine.go`, `internal/policy/engine_test.go`

- [ ] **Step 1: Write engine tests**

Test:
- NewEngine with valid policy file → loads rules correctly
- NewEngine with missing file → zero rules (no error)
- NewEngine with invalid YAML → error
- Evaluate with no rules → empty result, not blocked
- Evaluate with observe-mode rule that triggers → violations populated, blocked=false
- Evaluate with enforce-mode rule that triggers → blocked=true, stops after first enforce
- Evaluate with mixed observe+enforce → accumulates observe violations, stops at first enforce
- Hot reload: modify file → rules update
- Hot reload with invalid file → keeps previous rules
- Rules() returns correct summaries

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/policy/ -v
```

- [ ] **Step 3: Implement engine.go**

```go
type Engine struct {
    policyPath string
    store      *capture.Store
    calc       *cost.Calculator
    rules      []Rule
    webhook    *WebhookConfig
    mu         sync.RWMutex
    stopCh     chan struct{}
}

func NewEngine(policyPath string, store *capture.Store, calc *cost.Calculator) (*Engine, error)  // Note: returns error (unlike spec which shows no error return — the plan's version is correct because loadPolicies can fail)
func (e *Engine) Evaluate(ctx EvalContext) EvalResult
func (e *Engine) Start()   // starts file watcher goroutine (2s poll)
func (e *Engine) Stop()    // stops watcher
func (e *Engine) Rules() []RuleSummary
func (e *Engine) loadPolicies() error  // parse YAML, construct rules
```

loadPolicies:
- Read file, parse YAML into PolicyFile
- For each RuleConfig, construct the appropriate Rule implementation
- Validate all rules
- Mutex-lock, swap rules slice, unlock

Start:
- goroutine that polls os.Stat every 2 seconds
- on mtime change: call loadPolicies
- if loadPolicies fails: log error, keep previous rules

Evaluate:
- RLock the rules
- Iterate rules, call Evaluate on each
- Accumulate violations, stop on first enforce
- **If `len(violations) > 0`** (not just if blocked), call `go e.sendWebhook(violations, ctx)`. The sendWebhook function filters by the configured events list — this is the only place webhooks are triggered.
- Return result

- [ ] **Step 4: Run tests**

```bash
go test ./internal/policy/ -v -cover
```

Target: ≥90% coverage on engine.go

- [ ] **Step 5: Commit**

```bash
git add internal/policy/engine.go internal/policy/engine_test.go
git commit -m "feat(v2): add policy engine with YAML loading, evaluation, and hot reload"
```

---

## Task 5: Webhook Delivery

**Files:**
- Create: `internal/policy/webhook.go`, `internal/policy/webhook_test.go`

- [ ] **Step 1: Write webhook tests**

Test:
- sendWebhook with enforce violation and events=[enforce] → HTTP POST sent
- sendWebhook with observe violation and events=[enforce] → filtered out, no POST
- sendWebhook with events=[all] → always sends
- sendWebhook with no URL configured → no-op
- Verify payload structure matches spec
- Timeout: mock server that sleeps → webhook doesn't block

- [ ] **Step 2: Implement webhook.go**

```go
func (e *Engine) sendWebhook(violations []Violation, ctx EvalContext) {
    if e.webhook == nil || e.webhook.URL == "" { return }

    // Filter by events config
    shouldSend := false
    for _, v := range violations {
        for _, event := range e.webhook.Events {
            if event == "all" || event == v.Mode {
                shouldSend = true
                break
            }
        }
    }
    if !shouldSend { return }

    // Build payload, POST with 5s timeout
    // Fire-and-forget (already in goroutine)
    // Log errors to stderr
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/policy/ -v -cover
```

- [ ] **Step 4: Commit**

```bash
git add internal/policy/webhook.go internal/policy/webhook_test.go
git commit -m "feat(v2): add webhook delivery for policy violations"
```

---

## Task 6: Schema Migration & Store Updates

**Files:**
- Modify: `internal/capture/models.go`
- Modify: `internal/capture/store.go`

- [ ] **Step 1: Write tests for new store capabilities**

Test:
- InsertTrace with PolicyViolations → stored and retrieved correctly via GetTrace (round-trip)
- InsertTrace with PolicyViolations → retrieved correctly via ListTraces
- ListTraces with HasViolations=true → only returns traces with violations
- ListTraces with HasViolations=false/nil → returns all traces
- GetStats returns ViolationsByRule and ViolationsByMode (respecting time range)
- Schema migration adds policy_violations column and rate_limit_state table

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/capture/ -v
```

- [ ] **Step 3: Update models.go**

Add to Trace struct:
```go
PolicyViolations *string `json:"policy_violations"` // JSON array or null
```

Add to Stats struct:
```go
ViolationsByRule map[string]int         `json:"violations_by_rule"`
ViolationsByMode map[string]int         `json:"violations_by_mode"` // {"observe": N, "enforce": N}
```

- [ ] **Step 4: Update store.go**

Schema migration — add to the existing `migrate()` function:
```sql
ALTER TABLE traces ADD COLUMN policy_violations TEXT;

CREATE TABLE IF NOT EXISTS rate_limit_state (
    scope_key   TEXT PRIMARY KEY,
    window_start TEXT NOT NULL,
    count       INTEGER DEFAULT 0,
    updated_at  TEXT NOT NULL
);
```

Note: SQLite `ALTER TABLE ADD COLUMN` will fail if the column already exists. Use a check: query `PRAGMA table_info(traces)` and only add if not present. Or use `CREATE TABLE IF NOT EXISTS` pattern with a version check.

Update InsertTrace: include `policy_violations` in the INSERT.

**Update GetTrace:** Add `policy_violations` to the SELECT list and Scan call. The returned Trace struct must include the new field.

**Update ListTraces:** Same — add `policy_violations` to the SELECT list and Scan call. Without this, the API and dashboard will never see violation data.

Add `HasViolations *bool` to TraceFilter. In ListTraces, if `HasViolations` is true, add `AND policy_violations IS NOT NULL AND policy_violations != 'null'`.

Update GetStats to query violations — **must respect the same from/to time conditions** as the other queries:
```sql
SELECT policy_violations FROM traces WHERE policy_violations IS NOT NULL AND policy_violations != 'null'
-- Add: AND created_at >= ? AND created_at <= ? (using the same conditions/args from the existing time filter)
```
Parse each JSON array, count by rule name and by mode. Initialize `ViolationsByRule` and `ViolationsByMode` maps even if empty (avoid nil maps in JSON output).

- [ ] **Step 5: Run tests**

```bash
go test ./internal/capture/ -v -cover
```

- [ ] **Step 6: Commit**

```bash
git add internal/capture/
git commit -m "feat(v2): add policy_violations to trace schema and store"
```

---

## Task 7: Proxy Integration

**Files:**
- Modify: `internal/proxy/server.go`

- [ ] **Step 1: Write integration tests**

Test:
- Request with policy engine that has an enforce rule → returns provider-formatted error, status 400, X-Corvade-Policy-Violation header
- Request with observe-only violations → request still forwarded, trace has violations attached
- Request with PII redaction (action=redact) → modified body forwarded to upstream
- Request with no policy engine (nil) → v1 behavior unchanged
- Blocked trace captured with status_code=499

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/proxy/ -v
```

- [ ] **Step 3: Implement proxy changes**

Add to Server struct:
```go
policyEngine *policy.Engine
```

Add `SetPolicyEngine(e *policy.Engine)` method.

In handleProxy, after parsing request and extracting headers, before forwarding:
1. If policyEngine is nil, skip (v1 behavior)
2. Build EvalContext with provider, model, agent, session, step (from corvadeStep variable, already extracted), requestBody, token estimate, estimated cost
3. Call policyEngine.Evaluate
4. If blocked: writeProviderError, captureTrace with status 499, emit trace:blocked event, return
5. If modified (PII redaction): replace reqBody
6. Store violations for later capture

Update captureTrace signature to accept violations `[]policy.Violation`. Serialize as JSON string for the trace.

**CRITICAL: Atomic change required.** The `captureTrace` signature change and ALL its call sites must be updated in the same edit. The existing call on line ~213 of `handleProxy` must gain the new `violations` parameter (pass `nil` for the normal path). If you change the signature without updating all call sites, the build will break.

Add writeProviderError function that formats errors in OpenAI or Anthropic format based on provider. Sets `X-Corvade-Policy-Violation: true` header and status 400.

Emit `trace:blocked` WebSocket event for blocked requests (alongside `trace:new` for observed violations).

- [ ] **Step 4: Run tests**

```bash
go test ./internal/proxy/ -v -cover
```

- [ ] **Step 5: Run full test suite**

```bash
go test ./... -v
```

All must pass — no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/proxy/
git commit -m "feat(v2): integrate policy engine into proxy with enforce/observe/redact"
```

---

## Task 8: CLI — Policies Command + Tail Updates

**Files:**
- Create: `internal/cli/policies.go`, `internal/cli/policies_test.go`
- Modify: `internal/cli/tail.go`
- Modify: `internal/cli/start.go`
- Modify: `cmd/corvade/main.go`

- [ ] **Step 1: Write tests for policies command**

Test:
- `corvade policies` with valid file → lists rules with correct format
- `corvade policies validate` with valid file → exit 0, success message
- `corvade policies validate` with invalid file → exit 1, error messages
- Missing file → "No policies file found" message

- [ ] **Step 2: Implement policies.go**

Two subcommands:
- `corvade policies` (default): read `~/.corvade/policies.yaml`, parse, display table
- `corvade policies validate`: parse + validate, exit 0 or 1

Uses the policy engine's loading/validation logic but doesn't start the watcher or the proxy.

- [ ] **Step 3: Update tail.go**

Listen for both `trace:new` and `trace:blocked` WebSocket events.
- `trace:blocked`: print `⛨ BLOCKED: {rule_type} {message}`
- `trace:new` with `policy_violations` field: print `⚠ observe: {rule_type} ({detail})`
- Clean trace: unchanged

- [ ] **Step 4: Update start.go**

In the start command, after creating the store and cost calculator:
```go
policyPath := filepath.Join(os.Getenv("HOME"), ".corvade", "policies.yaml")
policyEngine, err := policy.NewEngine(policyPath, store, calc)
if err != nil {
    log.Printf("warning: failed to load policies: %v", err)
} else {
    policyEngine.Start()
    defer policyEngine.Stop()
    proxySrv.SetPolicyEngine(policyEngine)
}
```

Print policy count in the startup banner if policies are loaded.

- [ ] **Step 5: Update main.go**

Register the policies command:
```go
rootCmd.AddCommand(cli.NewPoliciesCmd())
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/cli/ -v
go test ./... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/cli/ cmd/corvade/
git commit -m "feat(v2): add corvade policies command and violation indicators in tail"
```

---

## Task 9: Dashboard — Policy Badge & Tab

**Files:**
- Create: `dashboard/src/components/PolicyBadge.tsx`, `dashboard/src/components/PolicyTab.tsx`
- Modify: `dashboard/src/components/Timeline.tsx`
- Modify: `dashboard/src/components/DetailInspector.tsx`

- [ ] **Step 1: Create PolicyBadge component**

Small shield icon SVG. Props: `{ mode: 'observe' | 'enforce' }`.
- observe: violet fill, small size (14px)
- enforce: red fill

- [ ] **Step 2: Create PolicyTab component**

Props: `{ violations: Violation[], blocked: boolean }`.
- If blocked: red banner "This request was blocked by policy enforcement"
- List each violation: rule name, type, mode badge (PolicyBadge), message
- Styled consistently with the existing detail inspector

- [ ] **Step 3: Update Timeline.tsx**

- Parse `policy_violations` from trace data (JSON string → array)
- Show PolicyBadge before the status code column for traces with violations
- Add a "Violations only" toggle button in the filter bar
- Blocked traces (status_code 499) get the existing `.row-error` treatment

- [ ] **Step 4: Update DetailInspector.tsx**

- Add "Policy" as a third tab alongside Request/Response
- Only show the Policy tab if `policy_violations` is non-null
- Render PolicyTab component with the parsed violations

- [ ] **Step 5: Verify dashboard builds**

```bash
cd dashboard && npm run build
```

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/
git commit -m "feat(v2): add policy badge and violations tab in dashboard"
```

---

## Task 10: Dashboard — Stats Violations Section

**Files:**
- Modify: `dashboard/src/app/stats/page.tsx`

- [ ] **Step 1: Add violations section to Stats page**

Below the existing "By Agent" breakdown, add:
- Section heading: "POLICY VIOLATIONS"
- "By Rule" bar chart: same CSS bar style as By Model/By Agent, using `violations_by_rule` from stats API
- "By Mode" breakdown: observe count vs enforce count from `violations_by_mode`
- Only show this section if there are violations (hide if all zeros)

- [ ] **Step 2: Verify dashboard builds**

```bash
cd dashboard && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/app/stats/page.tsx
git commit -m "feat(v2): add policy violations section to Stats tab"
```

---

## Task 11: Final Build, Integration Test & Rebuild

**Files:**
- Modify: `Makefile` (rebuild dashboard embed)

- [ ] **Step 1: Run full Go test suite**

```bash
go test ./... -v -cover
```

All tests pass. Check coverage — target ≥85% on new policy packages.

- [ ] **Step 2: Build dashboard**

```bash
cd dashboard && npm run build
```

- [ ] **Step 3: Rebuild Go binary**

```bash
make build
```

- [ ] **Step 4: End-to-end test**

Create a test policy file, start Corvade, send requests through the proxy, verify:
- Allowed requests pass through
- Blocked requests return provider-formatted errors
- Violations show in dashboard
- `corvade tail` shows violation indicators
- `corvade policies` lists rules

- [ ] **Step 5: Run full test suite one more time**

```bash
go test ./... -v
```

- [ ] **Step 6: Commit and push**

```bash
git add .
git commit -m "feat(v2): complete governance and policy enforcement"
git push origin development
```

---

## Summary

| Task | Description | Key Files |
|------|------------|-----------|
| 1 | Policy types & token estimation | internal/policy/types.go, tokens.go |
| 2 | Token limit + model allowlist rules | internal/policy/rules/ |
| 3 | Rate limit + PII redaction + cost ceiling rules | internal/policy/rules/ |
| 4 | Policy engine (load, watch, evaluate) | internal/policy/engine.go |
| 5 | Webhook delivery | internal/policy/webhook.go |
| 6 | Schema migration & store updates | internal/capture/ |
| 7 | Proxy integration | internal/proxy/server.go |
| 8 | CLI — policies command + tail updates | internal/cli/ |
| 9 | Dashboard — policy badge & tab | dashboard/src/components/ |
| 10 | Dashboard — stats violations section | dashboard/src/app/stats/ |
| 11 | Final build & integration test | Full rebuild |
