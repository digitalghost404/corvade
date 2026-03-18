package proxy

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// streamUsage holds token usage extracted from the final SSE chunk.
type streamUsage struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
}

// streamSSE reads an SSE upstream response, forwards each line immediately to w,
// and assembles the full response content for capture. It also extracts token
// usage from the final data chunk that OpenAI includes with usage information.
func streamSSE(upstream io.Reader, w http.ResponseWriter) (assembled string, usage *streamUsage, err error) {
	flusher, canFlush := w.(http.Flusher)

	var sb strings.Builder
	scanner := bufio.NewScanner(upstream)

	for scanner.Scan() {
		line := scanner.Text()

		// Forward the raw line to the client.
		_, writeErr := io.WriteString(w, line+"\n")
		if writeErr != nil {
			// Client disconnected; stop streaming.
			break
		}
		if canFlush {
			flusher.Flush()
		}

		// Only process "data: " prefixed lines.
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")

		// Skip the terminal sentinel.
		if payload == "[DONE]" {
			continue
		}

		// Try to parse the JSON payload.
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				PromptTokensDetails *struct {
					CachedTokens int `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
			} `json:"usage"`
		}

		if jsonErr := json.Unmarshal([]byte(payload), &chunk); jsonErr != nil {
			continue
		}

		// Accumulate content from all choices.
		for _, choice := range chunk.Choices {
			sb.WriteString(choice.Delta.Content)
		}

		// Capture usage when present (OpenAI sends it in the last data chunk).
		if chunk.Usage != nil {
			u := &streamUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
			}
			if chunk.Usage.PromptTokensDetails != nil {
				u.CachedTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			usage = u
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return sb.String(), usage, scanErr
	}

	return sb.String(), usage, nil
}
