package env

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeEnvProvider struct {
	Folder string
}

// Register registe a new function for make a service instance
func (provider *HadeEnvProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeEnv
}

// Boot will called when the service instantiate
func (provider *HadeEnvProvider) Boot(c framework.Container) error {
	app := c.MustMake(contract.AppKey).(contract.App)
	provider.Folder = app.BaseFolder()
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeEnvProvider) IsDefer() bool {
	return false
}

// Params define the necessary params for NewInstance
func (provider *HadeEnvProvider) Params(c framework.Container) []interface{} {
	return []interface{}{provider.Folder}
}

/// Name define the name for this service
func (provider *HadeEnvProvider) Name() string {
	return contract.EnvKey
}
