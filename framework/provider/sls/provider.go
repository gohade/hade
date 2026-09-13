package sls

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeSLSProvider struct {
}

// Register registe a new function for make a service instance
func (provider *HadeSLSProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeSLSService
}

// Boot will called when the service instantiate
func (provider *HadeSLSProvider) Boot(c framework.Container) error {
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeSLSProvider) IsDefer() bool {
	return true
}

// Params define the necessary params for NewInstance
func (provider *HadeSLSProvider) Params(c framework.Container) []interface{} {
	return []interface{}{c}
}

// Name define the name for this service
func (provider *HadeSLSProvider) Name() string {
	return contract.SLSKey
}
