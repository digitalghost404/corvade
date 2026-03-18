package cost

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCalculatorGPT4o(t *testing.T) {
	calc := NewCalculator(nil)
	// gpt-4o: input $0.0025 per 1K, output $0.01 per 1K
	// 1000 input tokens = $0.0025, 1000 output tokens = $0.01
	// Total = $0.0125
	cost := calc.Calculate("gpt-4o", 1000, 1000)
	expected := 0.0125
	if !almostEqual(cost, expected) {
		t.Errorf("gpt-4o: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorGPT4oMini(t *testing.T) {
	calc := NewCalculator(nil)
	// gpt-4o-mini: input $0.00015 per 1K, output $0.0006 per 1K
	// 1000 input tokens = $0.00015, 1000 output tokens = $0.0006
	// Total = $0.00075
	cost := calc.Calculate("gpt-4o-mini", 1000, 1000)
	expected := 0.00075
	if !almostEqual(cost, expected) {
		t.Errorf("gpt-4o-mini: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorGPT35Turbo(t *testing.T) {
	calc := NewCalculator(nil)
	// gpt-3.5-turbo: input $0.0005 per 1K, output $0.002 per 1K
	// 500 input tokens = $0.00025, 2000 output tokens = $0.004
	// Total = $0.00425
	cost := calc.Calculate("gpt-3.5-turbo", 500, 2000)
	expected := 0.00425
	if !almostEqual(cost, expected) {
		t.Errorf("gpt-3.5-turbo: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorClaude35Sonnet(t *testing.T) {
	calc := NewCalculator(nil)
	// claude-3-5-sonnet-20241022: input $0.003 per 1K, output $0.015 per 1K
	// 1000 input tokens = $0.003, 1000 output tokens = $0.015
	// Total = $0.018
	cost := calc.Calculate("claude-3-5-sonnet-20241022", 1000, 1000)
	expected := 0.018
	if !almostEqual(cost, expected) {
		t.Errorf("claude-3-5-sonnet-20241022: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorUnknownModel(t *testing.T) {
	calc := NewCalculator(nil)
	// Unknown model should return 0
	cost := calc.Calculate("unknown-model", 1000, 1000)
	expected := 0.0
	if !almostEqual(cost, expected) {
		t.Errorf("unknown model: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorWithOverrides(t *testing.T) {
	overrides := map[string]ModelPricing{
		"gpt-4o": {InputPer1K: 0.01, OutputPer1K: 0.02},
	}
	calc := NewCalculator(overrides)
	// With override: input $0.01 per 1K, output $0.02 per 1K
	// 1000 input tokens = $0.01, 1000 output tokens = $0.02
	// Total = $0.03
	cost := calc.Calculate("gpt-4o", 1000, 1000)
	expected := 0.03
	if !almostEqual(cost, expected) {
		t.Errorf("gpt-4o with override: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorOverrideDoesNotAffectOthers(t *testing.T) {
	overrides := map[string]ModelPricing{
		"gpt-4o": {InputPer1K: 0.01, OutputPer1K: 0.02},
	}
	calc := NewCalculator(overrides)
	// gpt-3.5-turbo should still use default pricing
	cost := calc.Calculate("gpt-3.5-turbo", 1000, 1000)
	expected := 0.0025 // $0.0005 * 1 + $0.002 * 1
	if !almostEqual(cost, expected) {
		t.Errorf("gpt-3.5-turbo with overrides: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorZeroTokens(t *testing.T) {
	calc := NewCalculator(nil)
	cost := calc.Calculate("gpt-4o", 0, 0)
	expected := 0.0
	if !almostEqual(cost, expected) {
		t.Errorf("zero tokens: expected %f, got %f", expected, cost)
	}
}

func TestCalculatorFractionalTokens(t *testing.T) {
	calc := NewCalculator(nil)
	// gpt-4o: input $0.0025 per 1K, output $0.01 per 1K
	// 500 input tokens = $0.00125, 500 output tokens = $0.005
	// Total = $0.00625
	cost := calc.Calculate("gpt-4o", 500, 500)
	expected := 0.00625
	if !almostEqual(cost, expected) {
		t.Errorf("fractional tokens: expected %f, got %f", expected, cost)
	}
}
