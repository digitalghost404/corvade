package policy

import (
	"encoding/json"
	"math"
	"strings"
)

// EstimateTokens estimates the token count for an LLM request body.
//
// It supports both OpenAI and Anthropic JSON formats:
//   - messages[].content  — string or array of content parts (only "text" parts counted)
//   - top-level "system"  — Anthropic system prompt string
//
// Algorithm: extract all text → count words → multiply by 1.3 → add max_tokens if present.
// Returns 0 for empty or unparseable input.
func EstimateTokens(body []byte) int {
	if len(body) == 0 {
		return 0
	}

	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		return 0
	}

	var sb strings.Builder

	// Anthropic top-level system prompt.
	if raw, ok := req["system"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && s != "" {
			sb.WriteString(s)
			sb.WriteByte(' ')
		}
	}

	// messages array — shared structure for OpenAI and Anthropic.
	if raw, ok := req["messages"]; ok {
		var messages []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &messages); err == nil {
			for _, msg := range messages {
				contentRaw, ok := msg["content"]
				if !ok {
					continue
				}
				extractContent(contentRaw, &sb)
			}
		}
	}

	text := sb.String()
	words := countWords(text)
	estimate := int(math.Round(float64(words) * 1.3))

	// Add max_tokens if present.
	if raw, ok := req["max_tokens"]; ok {
		var maxTok int
		if err := json.Unmarshal(raw, &maxTok); err == nil {
			estimate += maxTok
		}
	}

	return estimate
}

// extractContent handles content that is either a plain string or an array of parts.
func extractContent(raw json.RawMessage, sb *strings.Builder) {
	// Try plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		sb.WriteString(s)
		sb.WriteByte(' ')
		return
	}

	// Try array of content parts (OpenAI vision / tool-use format).
	var parts []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil {
		return
	}
	for _, part := range parts {
		typeRaw, hasType := part["type"]
		if !hasType {
			continue
		}
		var partType string
		if err := json.Unmarshal(typeRaw, &partType); err != nil || partType != "text" {
			continue
		}
		textRaw, ok := part["text"]
		if !ok {
			continue
		}
		var t string
		if err := json.Unmarshal(textRaw, &t); err == nil && t != "" {
			sb.WriteString(t)
			sb.WriteByte(' ')
		}
	}
}

// countWords splits text on whitespace and counts words.
func countWords(text string) int {
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
