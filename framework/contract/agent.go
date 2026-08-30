package contract

import (
	"context"

	"github.com/pkg/errors"
)

const AgentKey = "hade:agent"

const (
	EventSession     = "session"
	EventThought     = "thought"
	EventAction      = "action"
	EventObservation = "observation"
	EventFinal       = "final"
	EventError       = "error"
	EventDone        = "done"
	ContentMaxBytes  = 4096
	DefaultMaxIter   = 8
	// RequestBodyMaxBytes 是 Agent HTTP 接口允许的请求体最大字节数，只防入口撑爆，不限制 Session 历史。
	RequestBodyMaxBytes = 64<<10 + 1024
)

var (
	ErrSessionNotFound = errors.New("session_not_found")
	ErrSessionBusy     = errors.New("session_busy")
	ErrEmptyMessage    = errors.New("empty_message")
	ErrMaxIterations   = errors.New("max_iterations")
	ErrCanceled        = errors.New("canceled")
	ErrLLMFailed       = errors.New("llm_failed")
	// ErrInternal 表示 Agent 内部异常（含被 recover 的 panic），调用方应视为 500。
	ErrInternal = errors.New("internal")
)

// Session 表示 Agent 的对话会话，包括会话唯一 ID 及消息历史。
// - ID: 唯一标识 session，用于多轮会话和持久化。
// - Messages: 本 session 的完整消息历史，结构与 LLM Chat API 兼容。
// 实现应保证 Messages 顺序（从最早到当前）。
type Session struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

// ToolHandler 定义工具函数的签名。
// - ctx: 处理上下文。
// - argsJSON: JSON 字符串，表示调用参数（根据 ToolSpec 的 parameters 结构）。
// 返回：
// - observation: 工具执行结果，将存入 Agent session，作为 tool event 输出给 LLM。
// - err: 工具处理中的错误（如参数解析失败、执行故障等）。
type ToolHandler func(ctx context.Context, argsJSON string) (observation string, err error)

// AgentEvent 为 Agent 在推理过程中的事件。
// - Type: 事件类型（见常量 EventSession、EventThought、EventAction、EventObservation、EventFinal、EventError、EventDone）。
// - Data: 事件内容，结构依赖于不同事件类型，常见键有 message、tool、error 等。
type AgentEvent struct {
	Type string
	Data map[string]interface{}
}

// Agent 抽象定义 Agent 系统的核心功能接口。
// 任意实现必须保证并发安全；各接口含义如下：
// - CreateSession: 新建空白 session，返回唯一 sessionID。
// - GetSession: 读取指定 session 的完整信息（如消息历史）。
// - RegisterTool: 注册可调用工具，LLM 可在推理中请求该工具。重复注册同名工具应覆盖或忽略（见实现）。
// - ListTools: 返回当前注册的所有工具规范。
// - Run: 在指定 session 上追加 user 消息并启动推理流程。事件过程通过 events channel 逐步推送（含中间思考、工具使用、观察、最终回复、错误等）。
type Agent interface {
	CreateSession(ctx context.Context) (sessionID string, err error)
	GetSession(ctx context.Context, id string) (Session, error)
	RegisterTool(spec ToolSpec, handler ToolHandler)
	ListTools() []ToolSpec
	Run(ctx context.Context, sessionID, userMessage string, events chan<- AgentEvent) error
}
