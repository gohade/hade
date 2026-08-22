package contract

import "context"

const LLMKey = "hade:llm"

const (
	FinishStop      = "stop"
	FinishToolCalls = "tool_calls"
	FinishLength    = "length"
)

type Message struct {
	Role       string // system | user | assistant | tool
	Content    string
	ToolCallID string // role=tool 时对应的 call id
	ToolCalls  []ToolCall
}

type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema object
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON object as string
}

type ChatRequest struct {
	Messages    []Message
	Tools       []ToolSpec
	MaxTokens   int
	Temperature float32
}

type ChatResponse struct {
	Message   Message
	ToolCalls []ToolCall
	Finish    string
}

type LLM interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
