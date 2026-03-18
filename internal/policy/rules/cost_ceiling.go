package rules

import (
	"fmt"
	"sync"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/policy"
)

// CostCeilingRule tracks cumulative cost with cached DB queries and blocks
// requests when the spending ceiling would be exceeded.
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

const cacheTTL = 5 * time.Second

// NewCostCeilingRule constructs a CostCeilingRule from a raw config map.
// Required config keys: "max_cost" (number), "window" (duration string), "per" (agent|session|global).
// The store parameter must not be nil.
func NewCostCeilingRule(name, mode string, config map[string]interface{}, store *capture.Store) (*CostCeilingRule, error) {
	if store == nil {
		return nil, fmt.Errorf("cost_ceiling rule %q: store must not be nil", name)
	}

	raw, ok := config["max_cost"]
	if !ok {
		return nil, fmt.Errorf("cost_ceiling rule %q: missing required config key 'max_cost'", name)
	}
	var maxCost float64
	switch v := raw.(type) {
	case float64:
		maxCost = v
	case int:
		maxCost = float64(v)
	default:
		return nil, fmt.Errorf("cost_ceiling rule %q: 'max_cost' must be a number, got %T", name, raw)
	}

	windowRaw, ok := config["window"]
	if !ok {
		return nil, fmt.Errorf("cost_ceiling rule %q: missing required config key 'window'", name)
	}
	windowStr, ok := windowRaw.(string)
	if !ok {
		return nil, fmt.Errorf("cost_ceiling rule %q: 'window' must be a string, got %T", name, windowRaw)
	}

	windowDur, err := parseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("cost_ceiling rule %q: invalid window %q: %w", name, windowStr, err)
	}

	perRaw, ok := config["per"]
	if !ok {
		return nil, fmt.Errorf("cost_ceiling rule %q: missing required config key 'per'", name)
	}
	per, ok := perRaw.(string)
	if !ok {
		return nil, fmt.Errorf("cost_ceiling rule %q: 'per' must be a string, got %T", name, perRaw)
	}

	switch per {
	case "global", "agent", "session":
	default:
		return nil, fmt.Errorf("cost_ceiling rule %q: 'per' must be one of global, agent, session; got %q", name, per)
	}

	return &CostCeilingRule{
		name:      name,
		mode:      mode,
		maxCost:   maxCost,
		windowDur: windowDur,
		per:       per,
		store:     store,
		cache:     make(map[string]cachedSpend),
	}, nil
}

func (r *CostCeilingRule) Name() string { return r.name }
func (r *CostCeilingRule) Type() string { return "cost_ceiling" }
func (r *CostCeilingRule) Mode() string { return r.mode }

// Evaluate checks whether the estimated cost of the current request would push
// total spending over the configured ceiling.
func (r *CostCeilingRule) Evaluate(ctx policy.EvalContext) *policy.Violation {
	key := r.scopeKey(ctx)
	spent := r.getSpend(key)

	total := spent + ctx.EstimatedCost
	if total <= r.maxCost {
		return nil
	}

	return &policy.Violation{
		Rule: r.name,
		Type: r.Type(),
		Mode: r.mode,
		Message: fmt.Sprintf(
			"Cost ceiling exceeded: $%.2f spent + $%.2f estimated = $%.2f > $%.2f limit (%s: %s, window: %s)",
			spent, ctx.EstimatedCost, total, r.maxCost,
			r.per, key, formatDuration(r.windowDur),
		),
	}
}

func (r *CostCeilingRule) scopeKey(ctx policy.EvalContext) string {
	switch r.per {
	case "agent":
		return ctx.Agent
	case "session":
		return ctx.Session
	default:
		return "global"
	}
}

func (r *CostCeilingRule) getSpend(key string) float64 {
	now := time.Now()

	r.cacheMu.Lock()
	if cached, ok := r.cache[key]; ok && now.Sub(cached.fetchedAt) < cacheTTL {
		r.cacheMu.Unlock()
		return cached.amount
	}
	r.cacheMu.Unlock()

	// Query the database
	amount := r.querySpend(key)

	r.cacheMu.Lock()
	r.cache[key] = cachedSpend{amount: amount, fetchedAt: now}
	r.cacheMu.Unlock()

	return amount
}

func (r *CostCeilingRule) querySpend(key string) float64 {
	cutoff := time.Now().Add(-r.windowDur).UTC().Format(time.RFC3339Nano)

	var query string
	var args []interface{}

	switch r.per {
	case "agent":
		query = "SELECT COALESCE(SUM(cost), 0) FROM traces WHERE agent = ? AND created_at >= ?"
		args = []interface{}{key, cutoff}
	case "session":
		query = "SELECT COALESCE(SUM(cost), 0) FROM traces WHERE session_id = ? AND created_at >= ?"
		args = []interface{}{key, cutoff}
	default: // global
		query = "SELECT COALESCE(SUM(cost), 0) FROM traces WHERE created_at >= ?"
		args = []interface{}{cutoff}
	}

	var amount float64
	if err := r.store.DB().QueryRow(query, args...).Scan(&amount); err != nil {
		return 0
	}
	return amount
}
