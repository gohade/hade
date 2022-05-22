package engine

import (
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

type QueueEngine interface {
	QueueType() string
	Push(content []byte) error
	Pop() ([]byte, error)

	// Listen 传入的signal代表信号，需要支持的有:
	// 1 close
	// 2 restart
	Listen(signal chan int) (<-chan []byte, error)
}

func NewQueueEngine(container framework.Container, config string) (QueueEngine, error) {
	configService := container.MustMake(contract.ConfigKey).(contract.Config)
	configMap := configService.GetStringMapString(config)
	typ, ok := configMap["type"]
	if !ok {
		return nil, errors.New("type must be config")
	}
	switch typ {
	case "sync":
		return NewSyncEngine(context.Background(), configMap)
	}
}
