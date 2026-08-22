package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

var _ contract.Agent = (*MemoryAgent)(nil)

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
	if strings.TrimSpace(userMessage) == "" {
		return contract.ErrEmptyMessage
	}
	a.mu.RLock()
	s, ok := a.sess[sessionID]
	a.mu.RUnlock()
	if !ok {
		return contract.ErrSessionNotFound
	}
	if !s.mu.TryLock() {
		return contract.ErrSessionBusy
	}
	defer s.mu.Unlock()

	send := func(typ string, data map[string]interface{}) bool {
		if events == nil {
			return true
		}
		select {
		case events <- contract.AgentEvent{Type: typ, Data: data}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	emitCanceled := func() error {
		send(contract.EventError, map[string]interface{}{
			"code":    "canceled",
			"message": ctx.Err().Error(),
		})
		return contract.ErrCanceled
	}

	if !send(contract.EventSession, map[string]interface{}{"session_id": sessionID}) {
		return emitCanceled()
	}

	s.data.Messages = append(s.data.Messages, contract.Message{Role: "user", Content: userMessage})

	emitTrunc := func(typ, key, val string) bool {
		return send(typ, map[string]interface{}{key: truncate(val, contract.ContentMaxBytes)})
	}

	for i := 0; i < a.maxIter; i++ {
		if err := ctx.Err(); err != nil {
			return emitCanceled()
		}
		tools := a.ListTools()
		resp, err := a.llm.Chat(ctx, contract.ChatRequest{
			Messages: append([]contract.Message(nil), s.data.Messages...),
			Tools:    tools,
		})
		if err != nil {
			if !send(contract.EventError, map[string]interface{}{
				"code":    "llm_failed",
				"message": err.Error(),
			}) {
				return emitCanceled()
			}
			return contract.ErrLLMFailed
		}
		if !emitTrunc(contract.EventThought, "content", resp.Message.Content) {
			return emitCanceled()
		}

		if len(resp.ToolCalls) > 0 {
			asst := contract.Message{Role: "assistant", Content: resp.Message.Content, ToolCalls: resp.ToolCalls}
			s.data.Messages = append(s.data.Messages, asst)
			for _, tc := range resp.ToolCalls {
				argsMap := map[string]interface{}{}
				argsValid := json.Unmarshal([]byte(tc.Arguments), &argsMap) == nil
				if !argsValid {
					argsMap = map[string]interface{}{}
				}
				if !send(contract.EventAction, map[string]interface{}{
					"name":      tc.Name,
					"arguments": argsMap,
				}) {
					return emitCanceled()
				}
				obs, herr := a.execTool(ctx, tc.Name, tc.Arguments)
				if herr != nil {
					obs = herr.Error()
				}
				obs = truncate(obs, contract.ContentMaxBytes)
				if !argsValid {
					obs = truncate("invalid tool arguments: "+tc.Arguments+" ; "+obs, contract.ContentMaxBytes)
				}
				if !send(contract.EventObservation, map[string]interface{}{
					"name":    tc.Name,
					"content": obs,
				}) {
					return emitCanceled()
				}
				s.data.Messages = append(s.data.Messages, contract.Message{
					Role: "tool", Content: obs, ToolCallID: tc.ID,
				})
			}
			continue
		}
		s.data.Messages = append(s.data.Messages, contract.Message{Role: "assistant", Content: resp.Message.Content})
		if !emitTrunc(contract.EventFinal, "content", resp.Message.Content) {
			return emitCanceled()
		}
		return nil
	}
	if !send(contract.EventError, map[string]interface{}{
		"code":    "max_iterations",
		"message": "max iterations reached",
	}) {
		return emitCanceled()
	}
	return contract.ErrMaxIterations
}

func (a *MemoryAgent) execTool(ctx context.Context, name, argsJSON string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, t := range a.tools {
		if t.spec.Name == name {
			return t.handler(ctx, argsJSON)
		}
	}
	return "", errors.New("unknown tool: " + name)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
