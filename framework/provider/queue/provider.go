package queue

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeQueueProvider struct {
	c framework.Container
}

// Register registe a new function for make a service instance
func (provider *HadeQueueProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeQueueService
}

// Boot will called when the service instantiate
func (provider *HadeQueueProvider) Boot(c framework.Container) error {
	provider.c = c
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeQueueProvider) IsDefer() bool {
	return false
}

// Params define the necessary params for NewInstance
func (provider *HadeQueueProvider) Params(c framework.Container) []interface{} {
	return []interface{}{provider.c}
}

/// Name define the name for this service
func (provider *HadeQueueProvider) Name() string {
	return contract.QueueKey
}
