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

var (
	ErrSessionNotFound = errors.New("session_not_found")
	ErrSessionBusy     = errors.New("session_busy")
	ErrEmptyMessage    = errors.New("empty_message")
	ErrMaxIterations   = errors.New("max_iterations")
	ErrCanceled        = errors.New("canceled")
	ErrLLMFailed       = errors.New("llm_failed")
)

type Session struct {
	ID       string
	Messages []Message
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
