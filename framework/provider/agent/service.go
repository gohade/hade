package agent

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

type memorySession struct {
	data contract.Session
	mu   sync.Mutex
}

type registeredTool struct {
	spec    contract.ToolSpec
	handler contract.ToolHandler
}

type MemoryAgent struct {
	llm     contract.LLM
	maxIter int
	mu      sync.RWMutex
	sess    map[string]*memorySession
	tools   []registeredTool
}

func NewMemoryAgent(llm contract.LLM, maxIter int) *MemoryAgent {
	if maxIter <= 0 {
		maxIter = contract.DefaultMaxIter
	}
	return &MemoryAgent{
		llm:     llm,
		maxIter: maxIter,
		sess:    map[string]*memorySession{},
	}
}

func NewHadeAgentService(params ...interface{}) (interface{}, error) {
	llm := params[0].(contract.LLM)
	maxIter := params[1].(int)
	return NewMemoryAgent(llm, maxIter), nil
}

func (a *MemoryAgent) CreateSession(ctx context.Context) (string, error) {
	id := uuid.New().String()
	a.mu.Lock()
	a.sess[id] = &memorySession{data: contract.Session{ID: id}}
	a.mu.Unlock()
	return id, nil
}

func (a *MemoryAgent) GetSession(ctx context.Context, id string) (contract.Session, error) {
	a.mu.RLock()
	s, ok := a.sess[id]
	a.mu.RUnlock()
	if !ok {
		return contract.Session{}, contract.ErrSessionNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := append([]contract.Message(nil), s.data.Messages...)
	for i := range msgs {
		msgs[i].Content = truncate(msgs[i].Content, contract.ContentMaxBytes)
	}
	return contract.Session{ID: s.data.ID, Messages: msgs}, nil
}

func (a *MemoryAgent) RegisterTool(spec contract.ToolSpec, handler contract.ToolHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = append(a.tools, registeredTool{spec: spec, handler: handler})
}

func (a *MemoryAgent) ListTools() []contract.ToolSpec {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]contract.ToolSpec, len(a.tools))
	for i, t := range a.tools {
		out[i] = t.spec
	}
	return out
}

func (a *MemoryAgent) Run(ctx context.Context, sessionID, userMessage string, events chan<- contract.AgentEvent) error {
	return errors.New("not implemented")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
