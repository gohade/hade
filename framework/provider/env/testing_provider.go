package env

import (
    "github.com/gohade/hade/framework"
    "github.com/gohade/hade/framework/contract"
)

type HadeTestingEnvProvider struct {
    Folder string
}

// Register registe a new function for make a service instance
func (provider *HadeTestingEnvProvider) Register(c framework.Container) framework.NewInstance {
    return NewHadeTestingEnv
}

// Boot will called when the service instantiate
func (provider *HadeTestingEnvProvider) Boot(c framework.Container) error {
    return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeTestingEnvProvider) IsDefer() bool {
    return false
}

// Params define the necessary params for NewInstance
func (provider *HadeTestingEnvProvider) Params(c framework.Container) []interface{} {
    return []interface{}{}
}

/// Name define the name for this service
func (provider *HadeTestingEnvProvider) Name() string {
    return contract.EnvKey
}
