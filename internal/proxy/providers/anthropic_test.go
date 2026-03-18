package providers

import (
	"testing"
)

func TestAnthropicParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    *RequestInfo
		wantErr bool
	}{
		{
			name:    "basic request with stream",
			body:    []byte(`{"model":"claude-3-5-sonnet-20241022","messages":[],"stream":true}`),
			want:    &RequestInfo{Model: "claude-3-5-sonnet-20241022", Stream: true},
			wantErr: false,
		},
		{
			name:    "request without stream",
			body:    []byte(`{"model":"claude-3-opus-20250219","messages":[]}`),
			want:    &RequestInfo{Model: "claude-3-opus-20250219", Stream: false},
			wantErr: false,
		},
		{
			name:    "request with stream false",
			body:    []byte(`{"model":"claude-3-haiku-20250307","messages":[],"stream":false}`),
			want:    &RequestInfo{Model: "claude-3-haiku-20250307", Stream: false},
			wantErr: false,
		},
		{
			name:    "invalid json",
			body:    []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty json",
			body:    []byte(`{}`),
			want:    &RequestInfo{Model: "", Stream: false},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnthropicParseRequest(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnthropicParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Model != tt.want.Model {
					t.Errorf("AnthropicParseRequest() Model = %v, want %v", got.Model, tt.want.Model)
				}
				if got.Stream != tt.want.Stream {
					t.Errorf("AnthropicParseRequest() Stream = %v, want %v", got.Stream, tt.want.Stream)
				}
			}
		})
	}
}

func TestAnthropicParseResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    *ResponseInfo
		wantErr bool
	}{
		{
			name:    "response with usage",
			body:    []byte(`{"usage":{"input_tokens":15,"output_tokens":25,"cache_read_input_tokens":3}}`),
			want:    &ResponseInfo{PromptTokens: 15, CompletionTokens: 25, CachedTokens: 3},
			wantErr: false,
		},
		{
			name:    "response without cache read tokens",
			body:    []byte(`{"usage":{"input_tokens":10,"output_tokens":20}}`),
			want:    &ResponseInfo{PromptTokens: 10, CompletionTokens: 20, CachedTokens: 0},
			wantErr: false,
		},
		{
			name:    "response with all zeros",
			body:    []byte(`{"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0}}`),
			want:    &ResponseInfo{PromptTokens: 0, CompletionTokens: 0, CachedTokens: 0},
			wantErr: false,
		},
		{
			name:    "invalid json",
			body:    []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing usage",
			body:    []byte(`{}`),
			want:    &ResponseInfo{PromptTokens: 0, CompletionTokens: 0, CachedTokens: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnthropicParseResponse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnthropicParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.PromptTokens != tt.want.PromptTokens {
					t.Errorf("AnthropicParseResponse() PromptTokens = %v, want %v", got.PromptTokens, tt.want.PromptTokens)
				}
				if got.CompletionTokens != tt.want.CompletionTokens {
					t.Errorf("AnthropicParseResponse() CompletionTokens = %v, want %v", got.CompletionTokens, tt.want.CompletionTokens)
				}
				if got.CachedTokens != tt.want.CachedTokens {
					t.Errorf("AnthropicParseResponse() CachedTokens = %v, want %v", got.CachedTokens, tt.want.CachedTokens)
				}
			}
		})
	}
}

func TestAnthropicExtractToolUse(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want []ToolCall
	}{
		{
			name: "extract single tool use",
			body: []byte(`{"content":[{"type":"tool_use","id":"toolu_123","name":"search","input":{}}]}`),
			want: []ToolCall{
				{ID: "toolu_123", Name: "search", Arguments: "{}"},
			},
		},
		{
			name: "extract multiple tool uses",
			body: []byte(`{"content":[{"type":"tool_use","id":"toolu_1","name":"search","input":{}},{"type":"tool_use","id":"toolu_2","name":"browse","input":{"url":"example.com"}}]}`),
			want: []ToolCall{
				{ID: "toolu_1", Name: "search", Arguments: "{}"},
				{ID: "toolu_2", Name: "browse", Arguments: "{\"url\":\"example.com\"}"},
			},
		},
		{
			name: "mixed content with text and tool use",
			body: []byte(`{"content":[{"type":"text","text":"Hello"},{"type":"tool_use","id":"toolu_123","name":"search","input":{}}]}`),
			want: []ToolCall{
				{ID: "toolu_123", Name: "search", Arguments: "{}"},
			},
		},
		{
			name: "no tool uses",
			body: []byte(`{"content":[{"type":"text","text":"Hello"}]}`),
			want: []ToolCall{},
		},
		{
			name: "empty content",
			body: []byte(`{"content":[]}`),
			want: []ToolCall{},
		},
		{
			name: "no content",
			body: []byte(`{}`),
			want: []ToolCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnthropicExtractToolCalls(tt.body)
			if len(got) != len(tt.want) {
				t.Errorf("AnthropicExtractToolCalls() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, call := range got {
				if call.ID != tt.want[i].ID {
					t.Errorf("AnthropicExtractToolCalls()[%d] ID = %v, want %v", i, call.ID, tt.want[i].ID)
				}
				if call.Name != tt.want[i].Name {
					t.Errorf("AnthropicExtractToolCalls()[%d] Name = %v, want %v", i, call.Name, tt.want[i].Name)
				}
				if call.Arguments != tt.want[i].Arguments {
					t.Errorf("AnthropicExtractToolCalls()[%d] Arguments = %v, want %v", i, call.Arguments, tt.want[i].Arguments)
				}
			}
		})
	}
}
