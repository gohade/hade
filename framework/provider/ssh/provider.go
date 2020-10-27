package ssh

import (
	"hade/framework"
	"hade/framework/contract"
)

// HadeSSHProvider provide a App service, it must be singlton, and not delay
type HadeSSHProvider struct {
	Config *contract.SSHConfig
}

// Register registe a new function for make a service instance
func (provider *HadeSSHProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeSSH
}

// Boot will called when the service instantiate
func (provider *HadeSSHProvider) Boot(c framework.Container) error {
	if provider.Config == nil {
		config := c.MustMake(contract.ConfigKey).(contract.Config)
		if config.IsExist("ssh") {
			provider.Config = &contract.SSHConfig{
				User:     config.GetString("ssh.user"),
				Password: config.GetString("ssh.password"),
				Host:     config.GetString("ssh.host"),
				Port:     config.GetString("ssh.port"),
				RsaKey:   config.GetString("ssh.rsa_key"),
				Timeout:  config.GetInt("ssh.timeout"),
			}
		}
	}
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (provider *HadeSSHProvider) IsDefer() bool {
	return true
}

// Params define the necessary params for NewInstance
func (provider *HadeSSHProvider) Params() []interface{} {
	return []interface{}{provider.Config}
}

/// Name define the name for this service
func (provider *HadeSSHProvider) Name() string {
	return contract.SSHKey
}
