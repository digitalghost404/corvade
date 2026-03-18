package providers

// RequestInfo contains extracted information from a provider request
type RequestInfo struct {
	Model  string
	Stream bool
}

// ResponseInfo contains token usage information from a provider response
type ResponseInfo struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
}

// ToolCall represents a tool/function call in a provider response
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}
