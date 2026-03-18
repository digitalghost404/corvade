package policy

import (
	"encoding/json"
	"math"
	"testing"
)

// helper: build a minimal OpenAI request body
func openAIBody(t *testing.T, messages []map[string]interface{}, extra map[string]interface{}) []byte {
	t.Helper()
	req := map[string]interface{}{
		"model":    "gpt-4o",
		"messages": messages,
	}
	for k, v := range extra {
		req[k] = v
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal openai body: %v", err)
	}
	return b
}

// helper: build a minimal Anthropic request body
func anthropicBody(t *testing.T, system string, messages []map[string]interface{}, extra map[string]interface{}) []byte {
	t.Helper()
	req := map[string]interface{}{
		"model":    "claude-3-5-sonnet-20241022",
		"messages": messages,
	}
	if system != "" {
		req["system"] = system
	}
	for k, v := range extra {
		req[k] = v
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal anthropic body: %v", err)
	}
	return b
}

// wordCount splits text by whitespace and counts words — matches estimator logic.
func wordCount(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			inWord = false
		} else {
			if !inWord {
				count++
				inWord = true
			}
		}
	}
	return count
}

// expectedTokens computes the expected result for given text and max_tokens.
// Uses math.Round to match the implementation.
func expectedTokens(text string, maxTokens int) int {
	words := wordCount(text)
	est := int(math.Round(float64(words) * 1.3))
	return est + maxTokens
}

// TestEstimateTokens_OpenAI_ThreeMessages: OpenAI request with 3 string-content messages.
func TestEstimateTokens_OpenAI_ThreeMessages(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "system", "content": "You are a helpful assistant."},
		{"role": "user", "content": "What is the capital of France?"},
		{"role": "assistant", "content": "The capital of France is Paris."},
	}
	body := openAIBody(t, msgs, nil)
	got := EstimateTokens(body)

	combinedText := "You are a helpful assistant. What is the capital of France? The capital of France is Paris."
	want := expectedTokens(combinedText, 0)

	if got != want {
		t.Errorf("OpenAI 3 messages: got %d, want %d", got, want)
	}
	if got <= 0 {
		t.Errorf("expected positive token estimate, got %d", got)
	}
}

// TestEstimateTokens_Anthropic_SystemPlusMessages: Anthropic body with system + messages.
func TestEstimateTokens_Anthropic_SystemPlusMessages(t *testing.T) {
	system := "You are Claude, an AI assistant made by Anthropic."
	msgs := []map[string]interface{}{
		{"role": "user", "content": "Hello, how are you today?"},
		{"role": "assistant", "content": "I am doing well, thank you for asking."},
	}
	body := anthropicBody(t, system, msgs, nil)
	got := EstimateTokens(body)

	combinedText := system + " Hello, how are you today? I am doing well, thank you for asking."
	want := expectedTokens(combinedText, 0)

	if got != want {
		t.Errorf("Anthropic system+messages: got %d, want %d", got, want)
	}
}

// TestEstimateTokens_MaxTokensAdded: max_tokens field is added to estimate.
func TestEstimateTokens_MaxTokensAdded(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "Summarize this document for me please."},
	}
	const maxTok = 500
	body := openAIBody(t, msgs, map[string]interface{}{"max_tokens": maxTok})

	got := EstimateTokens(body)

	combinedText := "Summarize this document for me please."
	want := expectedTokens(combinedText, maxTok)

	if got != want {
		t.Errorf("max_tokens: got %d, want %d", got, want)
	}
}

// TestEstimateTokens_EmptyBody: empty byte slice returns 0.
func TestEstimateTokens_EmptyBody(t *testing.T) {
	got := EstimateTokens([]byte{})
	if got != 0 {
		t.Errorf("empty body: expected 0, got %d", got)
	}
}

// TestEstimateTokens_MalformedJSON: unparseable JSON returns 0.
func TestEstimateTokens_MalformedJSON(t *testing.T) {
	got := EstimateTokens([]byte(`{not valid json`))
	if got != 0 {
		t.Errorf("malformed JSON: expected 0, got %d", got)
	}
}

// TestEstimateTokens_ContentArray: OpenAI vision format — content is an array of parts.
func TestEstimateTokens_ContentArray(t *testing.T) {
	msgs := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Describe this image in detail."},
				{"type": "image_url", "image_url": map[string]interface{}{"url": "https://example.com/img.jpg"}},
			},
		},
	}
	body := openAIBody(t, msgs, nil)
	got := EstimateTokens(body)

	// Only the text part should count; the image_url part has no "text" field.
	combinedText := "Describe this image in detail."
	want := expectedTokens(combinedText, 0)

	if got != want {
		t.Errorf("content array: got %d, want %d", got, want)
	}
	if got <= 0 {
		t.Errorf("expected positive estimate for content array, got %d", got)
	}
}
