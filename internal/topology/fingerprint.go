package topology

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// FingerprintMessages returns a SHA256 hex digest of the messages array
// extracted from a request body JSON.
func FingerprintMessages(requestBody string) string {
	var req struct {
		Messages json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(requestBody), &req); err != nil || req.Messages == nil {
		return ""
	}
	h := sha256.Sum256(req.Messages)
	return fmt.Sprintf("%x", h)
}

// ContainsAssistantResponse checks whether requestB's messages contain
// identifying content from responseA's assistant message.
// It extracts a marker from responseA (assistant content or tool_call IDs)
// and checks if requestB contains that marker.
func ContainsAssistantResponse(requestB string, responseA string) bool {
	marker := extractResponseMarker(responseA)
	if marker == "" {
		return false
	}
	return strings.Contains(requestB, marker)
}

// extractResponseMarker pulls an identifying string from a response body.
// For OpenAI: tries assistant content first, then first tool_call ID.
// For Anthropic: tries text content first, then first tool_use ID.
func extractResponseMarker(responseBody string) string {
	// Try OpenAI format
	var openAI struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID string `json:"id"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(responseBody), &openAI); err == nil && len(openAI.Choices) > 0 {
		msg := openAI.Choices[0].Message
		if msg.Content != "" {
			// Use a substring long enough to be identifying
			content := msg.Content
			if len(content) > 50 {
				content = content[:50]
			}
			return content
		}
		if len(msg.ToolCalls) > 0 && msg.ToolCalls[0].ID != "" {
			return msg.ToolCalls[0].ID
		}
	}

	// Try Anthropic format
	var anthropic struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			ID   string `json:"id"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(responseBody), &anthropic); err == nil {
		for _, block := range anthropic.Content {
			if block.Type == "text" && block.Text != "" {
				text := block.Text
				if len(text) > 50 {
					text = text[:50]
				}
				return text
			}
		}
		for _, block := range anthropic.Content {
			if block.Type == "tool_use" && block.ID != "" {
				return block.ID
			}
		}
	}

	return ""
}
