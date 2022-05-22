package trace

import (
    "github.com/gohade/hade/framework"
    "github.com/gohade/hade/framework/contract"
)

type HadeEventProvider struct {
    c framework.Container
}

// Register registe a new function for make a service instance
func (provider *HadeEventProvider) Register(c framework.Container) framework.NewInstance {
    return NewHadeEventService
}

// Boot will called when the service instantiate
func (provider *HadeEventProvider) Boot(c framework.Container) error {
    provider.c = c
    return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeEventProvider) IsDefer() bool {
    return false
}

// Params define the necessary params for NewInstance
func (provider *HadeEventProvider) Params(c framework.Container) []interface{} {
    return []interface{}{provider.c}
}

/// Name define the name for this service
func (provider *HadeEventProvider) Name() string {
    return contract.EventKey
}
