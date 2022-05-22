package engine

import (
	"container/list"
	"context"
)

type SyncEngine struct {
	queue *list.List
}

func (s *SyncEngine) QueueType() string {
	return "sync"
}

func (s *SyncEngine) Push(content []byte) error {
	s.queue.PushBack(content)
	return nil
}

func (s *SyncEngine) Pop() ([]byte, error) {
	if s.queue.Len() > 1 {
		front := s.queue.Front()
		s.queue.Remove(front)
		return front.Value.([]byte), nil
	}
	return nil, nil
}

func NewSyncEngine(ctx context.Context, config map[string]string) (*SyncEngine, error) {
	queue := list.New()
	return &SyncEngine{queue: queue}, nil
}
