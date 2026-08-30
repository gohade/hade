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
