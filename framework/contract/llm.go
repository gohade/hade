package contract

import "context"

const LLMKey = "hade:llm"

const (
	FinishStop      = "stop"
	FinishToolCalls = "tool_calls"
	FinishLength    = "length"
)

// Message 代表一次对话消息，支持标准 LLM 多角色格式。
// - role: system（系统提示）、user（用户输入）、assistant（助手回复）、tool（工具返回）。
// - content: 消息文本内容。role=tool 时可为空（工具输出通常通过 ToolCallID 匹配）。
// - tool_call_id: 仅 role=tool 时有值，指明属于哪个工具调用实例。
// - tool_calls: 仅 role=assistant 时带有，表示当前回复请求调用哪些工具（OpenAI/Open-Function 规范）。
type Message struct {
	Role       string     `json:"role"`                   // system | user | assistant | tool
	Content    string     `json:"content"`                // 消息内容文本。role=tool 时通常为空
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具输出所属的调用ID，仅 role=tool 时使用
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 本条消息需由助手请求工具调用时使用，仅 role=assistant 时出现
}

// ToolSpec 描述 LLM 支持调用的工具（即函数）的规范。
// - name: 唯一标识符。
// - description: 工具的用途简介（帮助 LLM 选择何时调用）。
// - parameters: JSON Schema 格式定义输入参数结构（兼容 OpenAI Function Calling）。
type ToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema object，定义工具调用入参
}

// ToolCall 表示助手一次工具调用的请求。
// - id: 调用实例唯一 ID（由 LLM 生成，后续工具返回时需要对应）。
// - name: 工具名。
// - arguments: JSON 格式的参数字符串，需按 ToolSpec 的定义解析。
type ToolCall struct {
	ID        string `json:"id"`        // 工具调用实例ID
	Name      string `json:"name"`      // 工具名称
	Arguments string `json:"arguments"` // 以字符串形式编码的JSON参数
}

// ChatRequest 表示与 LLM 进行对话时的请求结构体。
//
// - Messages: 历史对话消息。通常包含 system、user、assistant、tool 等多角色排列，LLM 需基于此生成回复。
// - Tools: 可选，向 LLM 暴露的工具规范（如函数调用等）。为空时表示不开放工具能力。
// - MaxTokens: 可选，本轮生成的输出 token 上限，超过后须截断；0 表示不限。
// - Temperature: 可选，采样温度，越高生成内容多样性越强，常用范围 0-2。
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
