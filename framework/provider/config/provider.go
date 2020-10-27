package config

import (
	"hade/framework"
	"hade/framework/contract"
)

type HadeConfigProvider struct {
	c      framework.Container
	folder string
	env    string

	envMaps map[string]string
}

// Register registe a new function for make a service instance
func (provider *HadeConfigProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeConfig
}

// Boot will called when the service instantiate
func (provider *HadeConfigProvider) Boot(c framework.Container) error {
	provider.folder = c.MustMake(contract.AppKey).(contract.App).ConfigPath()
	provider.envMaps = c.MustMake(contract.EnvKey).(contract.Env).All()
	provider.env = c.MustMake(contract.EnvKey).(contract.Env).AppEnv()
	provider.c = c
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeConfigProvider) IsDefer() bool {
	return true
}

// Params define the necessary params for NewInstance
func (provider *HadeConfigProvider) Params() []interface{} {
	return []interface{}{provider.folder, provider.envMaps, provider.env, provider.c}
}

/// Name define the name for this service
func (provider *HadeConfigProvider) Name() string {
	return contract.ConfigKey
}
