package kernel

import (
	"hade/framework"
	"hade/framework/contract"
	"hade/framework/gin"
)

// HadeAppProvider provide a App service, it must be singlton, and not delay
type HadeKernelProvider struct {
	HttpEngine *gin.Engine
}

// Register registe a new function for make a service instance
func (provider *HadeKernelProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeKernelService
}

// Boot will called when the service instantiate
func (provider *HadeKernelProvider) Boot(c framework.Container) error {
	if provider.HttpEngine == nil {
		provider.HttpEngine = gin.Default()
	}
	provider.HttpEngine.SetContainer(c.(*framework.HadeContainer))
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeKernelProvider) IsDefer() bool {
	return false
}

// Params define the necessary params for NewInstance
func (provider *HadeKernelProvider) Params() []interface{} {
	return []interface{}{provider.HttpEngine}
}

/// Name define the name for this service
func (provider *HadeKernelProvider) Name() string {
	return contract.KernelKey
}
