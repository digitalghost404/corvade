package topology

import "encoding/json"

// ExtractToolCallIDs returns tool_call IDs from a response body.
// Supports OpenAI format (choices[0].message.tool_calls[].id) and
// Anthropic format (content[].id where type="tool_use").
func ExtractToolCallIDs(responseBody string) []string {
	var ids []string

	// Try OpenAI format: choices[].message.tool_calls[].id
	var openAI struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID string `json:"id"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(responseBody), &openAI); err == nil {
		for _, c := range openAI.Choices {
			for _, tc := range c.Message.ToolCalls {
				if tc.ID != "" {
					ids = append(ids, tc.ID)
				}
			}
		}
	}
	if len(ids) > 0 {
		return ids
	}

	// Try Anthropic format: content[].id where type="tool_use"
	var anthropic struct {
		Content []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(responseBody), &anthropic); err == nil {
		for _, block := range anthropic.Content {
			if block.Type == "tool_use" && block.ID != "" {
				ids = append(ids, block.ID)
			}
		}
	}

	return ids
}

// ExtractReferencedToolCallIDs returns tool_call_ids referenced in request
// messages where role="tool".
func ExtractReferencedToolCallIDs(requestBody string) []string {
	var req struct {
		Messages []struct {
			Role       string `json:"role"`
			ToolCallID string `json:"tool_call_id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(requestBody), &req); err != nil {
		return nil
	}

	var ids []string
	for _, m := range req.Messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			ids = append(ids, m.ToolCallID)
		}
	}
	return ids
}
