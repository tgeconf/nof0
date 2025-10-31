package llm

// ChatRequest describes a single LLM chat invocation.
type ChatRequest struct {
	Model          string          `json:"model,omitempty"`
	Messages       []Message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	TopP           *float64        `json:"top_p,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Message represents a chat message in the conversation.
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ResponseFormat controls the structure of the assistant response.
type ResponseFormat struct {
	Type        string      `json:"type"`
	Schema      interface{} `json:"schema,omitempty"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Strict      *bool       `json:"strict,omitempty"`
}

// ChatResponse captures a non-streaming completion result.
type ChatResponse struct {
	ID          string   `json:"id"`
	Model       string   `json:"model"`
	Choices     []Choice `json:"choices"`
	Usage       Usage    `json:"usage"`
	Created     int64    `json:"created"`
	RawJSON     string   `json:"raw_json,omitempty"`
	Tier        string   `json:"tier,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int        `json:"index"`
	Message      Message    `json:"message"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall describes an assistant tool invocation.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function,omitempty"`
}

// FunctionCall holds structured function call data.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Usage summarises token accounting for a completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamResponse represents a streaming completion chunk.
type StreamResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Created int64          `json:"created"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice contains the delta for a single streaming choice.
type StreamChoice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta describes the incremental update in a streaming response.
type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
