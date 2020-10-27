package gorm

import (
	"hade/framework"
	"hade/framework/contract"
)

type GormServiceProvider struct {
	Config map[string]string
	framework.ServiceProvider
}

func (sp *GormServiceProvider) Name() string {
	return "gorm"
}

func (sp *GormServiceProvider) Register(c framework.Container) framework.NewInstance {
	return NewGormDB
}

func (sp *GormServiceProvider) IsDefer() bool {
	return true
}

func (sp *GormServiceProvider) Params() []interface{} {
	return []interface{}{sp.Config}
}

func (sp *GormServiceProvider) Boot(c framework.Container) error {
	if sp.Config == nil {
		if c.IsBind(contract.ConfigKey) {
			config := c.MustMake(contract.ConfigKey).(contract.Config)
			if config.IsExist("database.default") {
				sp.Config = config.GetStringMapString("database.default")
			}
		}
	}
	return nil
}
