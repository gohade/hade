package orm

import (
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

// GetBaseConfig 读取database.yaml根目录结构
func GetBaseConfig(c framework.Container) *contract.DBConfig {
	configService := c.MustMake(contract.ConfigKey).(contract.Config)
	logService := c.MustMake(contract.LogKey).(contract.Log)
	config := &contract.DBConfig{}
	// 直接使用配置服务的load方法读取,yaml文件
	err := configService.Load("database", config)
	if err != nil {
		// 直接使用logService来打印错误信息
		logService.Error(context.Background(), "parse database config error", nil)
		return nil
	}
	return config
}

// WithConfigPath 加载配置文件地址
func WithConfigPath(configPath string) contract.DBOption {
	return func(container framework.Container, config *contract.DBConfig) error {
		configService := container.MustMake(contract.ConfigKey).(contract.Config)
		// 加载configPath配置路径
		if err := configService.Load(configPath, config); err != nil {
			return err
		}
		return nil
	}
}

// WithGormConfig 表示自行配置Gorm的配置信息
func WithGormConfig(f func(options *contract.DBConfig)) contract.DBOption {
	return func(container framework.Container, config *contract.DBConfig) error {
        f(config)
        return nil
	}
}


// WithDryRun 设置空跑模式
func WithDryRun() contract.DBOption {
	return func(container framework.Container, config *contract.DBConfig) error {
		config.DryRun = true
		return nil
	}
}

// WithFullSaveAssociations 设置保存时候关联
func WithFullSaveAssociations() contract.DBOption {
	return func(container framework.Container, config *contract.DBConfig) error {
		config.FullSaveAssociations = true
		return nil
	}
}
