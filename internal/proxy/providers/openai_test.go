package providers

import (
	"testing"
)

func TestOpenAIParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    *RequestInfo
		wantErr bool
	}{
		{
			name:    "basic request with stream",
			body:    []byte(`{"model":"gpt-4o","messages":[],"stream":true}`),
			want:    &RequestInfo{Model: "gpt-4o", Stream: true},
			wantErr: false,
		},
		{
			name:    "request without stream",
			body:    []byte(`{"model":"gpt-4-turbo","messages":[]}`),
			want:    &RequestInfo{Model: "gpt-4-turbo", Stream: false},
			wantErr: false,
		},
		{
			name:    "request with stream false",
			body:    []byte(`{"model":"gpt-3.5-turbo","messages":[],"stream":false}`),
			want:    &RequestInfo{Model: "gpt-3.5-turbo", Stream: false},
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
			got, err := OpenAIParseRequest(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenAIParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Model != tt.want.Model {
					t.Errorf("OpenAIParseRequest() Model = %v, want %v", got.Model, tt.want.Model)
				}
				if got.Stream != tt.want.Stream {
					t.Errorf("OpenAIParseRequest() Stream = %v, want %v", got.Stream, tt.want.Stream)
				}
			}
		})
	}
}

func TestOpenAIParseResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    *ResponseInfo
		wantErr bool
	}{
		{
			name:    "response with usage",
			body:    []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20,"cached_tokens":5}}`),
			want:    &ResponseInfo{PromptTokens: 10, CompletionTokens: 20, CachedTokens: 5},
			wantErr: false,
		},
		{
			name:    "response without cached tokens",
			body:    []byte(`{"usage":{"prompt_tokens":15,"completion_tokens":25}}`),
			want:    &ResponseInfo{PromptTokens: 15, CompletionTokens: 25, CachedTokens: 0},
			wantErr: false,
		},
		{
			name:    "response with all zeros",
			body:    []byte(`{"usage":{"prompt_tokens":0,"completion_tokens":0,"cached_tokens":0}}`),
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
			got, err := OpenAIParseResponse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenAIParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.PromptTokens != tt.want.PromptTokens {
					t.Errorf("OpenAIParseResponse() PromptTokens = %v, want %v", got.PromptTokens, tt.want.PromptTokens)
				}
				if got.CompletionTokens != tt.want.CompletionTokens {
					t.Errorf("OpenAIParseResponse() CompletionTokens = %v, want %v", got.CompletionTokens, tt.want.CompletionTokens)
				}
				if got.CachedTokens != tt.want.CachedTokens {
					t.Errorf("OpenAIParseResponse() CachedTokens = %v, want %v", got.CachedTokens, tt.want.CachedTokens)
				}
			}
		})
	}
}

func TestOpenAIExtractToolCalls(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want []ToolCall
	}{
		{
			name: "extract single tool call",
			body: []byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_123","function":{"name":"search","arguments":"{}"}}]}}]}`),
			want: []ToolCall{
				{ID: "call_123", Name: "search", Arguments: "{}"},
			},
		},
		{
			name: "extract multiple tool calls",
			body: []byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_1","function":{"name":"search","arguments":"{}"}},{"id":"call_2","function":{"name":"browse","arguments":"{\"url\":\"example.com\"}"}}]}}]}`),
			want: []ToolCall{
				{ID: "call_1", Name: "search", Arguments: "{}"},
				{ID: "call_2", Name: "browse", Arguments: "{\"url\":\"example.com\"}"},
			},
		},
		{
			name: "no tool calls",
			body: []byte(`{"choices":[{"message":{"content":"Hello"}}]}`),
			want: []ToolCall{},
		},
		{
			name: "empty choices",
			body: []byte(`{"choices":[]}`),
			want: []ToolCall{},
		},
		{
			name: "no choices",
			body: []byte(`{}`),
			want: []ToolCall{},
		},
		{
			name: "invalid json returns empty slice",
			body: []byte(`{invalid}`),
			want: []ToolCall{},
		},
		{
			name: "choices with no tool calls returns nil",
			body: []byte(`{"choices":[{"message":{}}]}`),
			want: []ToolCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OpenAIExtractToolCalls(tt.body)
			if len(got) != len(tt.want) {
				t.Errorf("OpenAIExtractToolCalls() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, call := range got {
				if call.ID != tt.want[i].ID {
					t.Errorf("OpenAIExtractToolCalls()[%d] ID = %v, want %v", i, call.ID, tt.want[i].ID)
				}
				if call.Name != tt.want[i].Name {
					t.Errorf("OpenAIExtractToolCalls()[%d] Name = %v, want %v", i, call.Name, tt.want[i].Name)
				}
				if call.Arguments != tt.want[i].Arguments {
					t.Errorf("OpenAIExtractToolCalls()[%d] Arguments = %v, want %v", i, call.Arguments, tt.want[i].Arguments)
				}
			}
		})
	}
}
