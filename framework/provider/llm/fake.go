package llm

import (
	"context"
	"sync"

	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

type ScriptLLM struct {
	mu        sync.Mutex
	Responses []contract.ChatResponse
	Errs      []error
	Calls     []contract.ChatRequest
	idx       int
}

func (s *ScriptLLM) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.ChatResponse{}, err
	}
	s.Calls = append(s.Calls, req)
	i := s.idx
	s.idx++
	if i < len(s.Errs) && s.Errs[i] != nil {
		return contract.ChatResponse{}, s.Errs[i]
	}
	if i >= len(s.Responses) {
		return contract.ChatResponse{}, errors.New("llm script exhausted")
	}
	return s.Responses[i], nil
}
