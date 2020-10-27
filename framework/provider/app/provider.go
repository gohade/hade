package app

import (
	"hade/framework"
	"hade/framework/contract"
)

// HadeAppProvider provide a App service, it must be singlton, and not delay
type HadeAppProvider struct {
	app *HadeApp

	BasePath string
}

// Register registe a new function for make a service instance
func (provider *HadeAppProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeApp
}

// Boot will called when the service instantiate
func (provider *HadeAppProvider) Boot(c framework.Container) error {
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeAppProvider) IsDefer() bool {
	return false
}

// Params define the necessary params for NewInstance
func (provider *HadeAppProvider) Params() []interface{} {
	return []interface{}{provider.BasePath}
}

/// Name define the name for this service
func (provider *HadeAppProvider) Name() string {
	return contract.AppKey
}
