package providers

import (
	"encoding/json"
)

// OpenAIParseRequest extracts model and stream from an OpenAI API request body
func OpenAIParseRequest(body []byte) (*RequestInfo, error) {
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

// OpenAIParseResponse extracts token usage from an OpenAI API response body
func OpenAIParseResponse(body []byte) (*ResponseInfo, error) {
	var payload struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			CachedTokens     int `json:"cached_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &ResponseInfo{
		PromptTokens:     payload.Usage.PromptTokens,
		CompletionTokens: payload.Usage.CompletionTokens,
		CachedTokens:     payload.Usage.CachedTokens,
	}, nil
}

// OpenAIExtractToolCalls extracts tool calls from an OpenAI API response body
func OpenAIExtractToolCalls(body []byte) []ToolCall {
	var payload struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return []ToolCall{}
	}

	var calls []ToolCall
	if len(payload.Choices) == 0 {
		return calls
	}

	for _, tc := range payload.Choices[0].Message.ToolCalls {
		calls = append(calls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return calls
}
