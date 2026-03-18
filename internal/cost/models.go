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
