package contract

import "context"

const LLMKey = "hade:llm"

const (
	FinishStop      = "stop"
	FinishToolCalls = "tool_calls"
	FinishLength    = "length"
)

type Message struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // role=tool 时对应的 call id
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // role=assistant 时请求的工具调用
}

type ToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema object
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON object as string
}

type ChatRequest struct {
	Messages    []Message  `json:"messages"`
	Tools       []ToolSpec `json:"tools,omitempty"`
	MaxTokens   int        `json:"max_tokens,omitempty"`
	Temperature float32    `json:"temperature,omitempty"`
}

// ChatResponse 是 LLM 单轮响应。
//
// Message 是唯一权威载体：工具调用只从 Message.ToolCalls 读取，
// 不存在与之并列的顶层 ToolCalls 字段，避免两处语义不一致。
type ChatResponse struct {
	Message Message `json:"message"`
	Finish  string  `json:"finish"`
}

type LLM interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
