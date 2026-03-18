package policy

// Violation describes a single policy rule breach.
type Violation struct {
	Rule    string `json:"rule"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
	Message string `json:"message"`
}

// EvalContext holds everything needed to evaluate a request against policy rules.
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

// EvalResult is the output of evaluating all rules against an EvalContext.
type EvalResult struct {
	Violations []Violation
	Blocked    bool
	Modified   []byte
}

// Rule is the interface every policy rule must implement.
type Rule interface {
	Name() string
	Type() string
	Mode() string
	Evaluate(ctx EvalContext) *Violation
}

// RuleSummary is a serialisable summary of a rule's identity and validity.
type RuleSummary struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Mode  string `json:"mode"`
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// PolicyFile is the YAML schema for a policy configuration file.
type PolicyFile struct {
	Webhook *WebhookConfig `yaml:"webhook"`
	Rules   []RuleConfig   `yaml:"rules"`
}

// WebhookConfig defines the optional outbound webhook for policy events.
type WebhookConfig struct {
	URL    string   `yaml:"url"`
	Events []string `yaml:"events"`
}

// RuleConfig is the raw YAML representation of a single rule entry.
type RuleConfig struct {
	Name   string                 `yaml:"name"`
	Type   string                 `yaml:"type"`
	Mode   string                 `yaml:"mode"`
	Config map[string]interface{} `yaml:"config"`
}
