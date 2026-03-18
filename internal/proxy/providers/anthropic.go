package providers

import (
	"encoding/json"
)

// AnthropicParseRequest extracts model and stream from an Anthropic API request body
func AnthropicParseRequest(body []byte) (*RequestInfo, error) {
	var payload struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &RequestInfo{
		Model:  payload.Model,
		Stream: payload.Stream,
	}, nil
}

// AnthropicParseResponse extracts token usage from an Anthropic API response body
// Maps input_tokens to PromptTokens, output_tokens to CompletionTokens, and cache_read_input_tokens to CachedTokens
func AnthropicParseResponse(body []byte) (*ResponseInfo, error) {
	var payload struct {
		Usage struct {
			InputTokens            int `json:"input_tokens"`
			OutputTokens           int `json:"output_tokens"`
			CacheReadInputTokens   int `json:"cache_read_input_tokens"`
			CacheCreateInputTokens int `json:"cache_create_input_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &ResponseInfo{
		PromptTokens:     payload.Usage.InputTokens,
		CompletionTokens: payload.Usage.OutputTokens,
		CachedTokens:     payload.Usage.CacheReadInputTokens,
	}, nil
}

// AnthropicExtractToolCalls extracts tool use blocks from an Anthropic API response body
func AnthropicExtractToolCalls(body []byte) []ToolCall {
	var payload struct {
		Content []struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return []ToolCall{}
	}

	var calls []ToolCall
	for _, block := range payload.Content {
		if block.Type == "tool_use" {
			calls = append(calls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}

	return calls
}
