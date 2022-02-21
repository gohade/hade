package log

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeTestingLogProvider struct {
}

// Register registe a new function for make a service instance
func (provider *HadeTestingLogProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeTestingLog
}

// Boot will called when the service instantiate
func (provider *HadeTestingLogProvider) Boot(c framework.Container) error {
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeTestingLogProvider) IsDefer() bool {
	return false
}

// Params define the necessary params for NewInstance
func (provider *HadeTestingLogProvider) Params(c framework.Container) []interface{} {
	return []interface{}{}
}

/// Name define the name for this service
func (provider *HadeTestingLogProvider) Name() string {
	return contract.LogKey
}
