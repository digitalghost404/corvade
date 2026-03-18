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
