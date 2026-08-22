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
)

// 有界资源默认上限。Agent 是长驻内存服务，这些常量保证单进程内存占用可预估。
const (
	// DefaultMaxSessions 是单个 Agent 实例允许并存的 Session 数量上限。
	DefaultMaxSessions = 1000
	// DefaultMaxMessageBytes 是单条 user 消息允许的最大字节数。
	DefaultMaxMessageBytes = 64 << 10
	// DefaultMaxHistoryBytes 是单个 Session 历史允许占用的最大字节数。
	DefaultMaxHistoryBytes = 1 << 20
	// RequestBodyMaxBytes 是 Agent HTTP 接口允许的请求体最大字节数。
	//
	// 必须严格大于 DefaultMaxMessageBytes，否则一条刚好超过消息上限的请求会先被
	// 请求体上限挡掉，ErrMessageTooLarge 在 HTTP 路径上永远不可达。留 1KiB 给
	// JSON 信封与转义开销。
	RequestBodyMaxBytes = DefaultMaxMessageBytes + 1024
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
	// ErrSessionLimit 表示 Session 数量已达上限，无法再创建。
	ErrSessionLimit = errors.New("session_limit")
	// ErrMessageTooLarge 表示单条消息超过长度上限。
	ErrMessageTooLarge = errors.New("message_too_large")
	// ErrHistoryLimit 表示 Session 历史字节数已达上限。
	ErrHistoryLimit = errors.New("history_limit")
)

type Session struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

type ToolHandler func(ctx context.Context, argsJSON string) (observation string, err error)

type AgentEvent struct {
	Type string
	Data map[string]interface{}
}

type Agent interface {
	CreateSession(ctx context.Context) (sessionID string, err error)
	GetSession(ctx context.Context, id string) (Session, error)
	RegisterTool(spec ToolSpec, handler ToolHandler)
	ListTools() []ToolSpec
	Run(ctx context.Context, sessionID, userMessage string, events chan<- AgentEvent) error
}
