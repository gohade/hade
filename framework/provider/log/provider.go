package log

import (
	"github.com/gohade/hade/framework/provider/log/formatter"
	"io"
	"strings"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/log/services"
)

// HadeLogServiceProvider 服务提供者
type HadeLogServiceProvider struct {
	framework.ServiceProvider

	Driver string // Driver

	// 日志级别
	Level contract.LogLevel
	// 日志输出格式方法
	Formatter contract.Formatter
	// 日志context上下文信息获取函数
	CtxFielder contract.CtxFielder
	// 日志输出信息
	Output io.Writer
}

// Register 注册一个服务实例
func (l *HadeLogServiceProvider) Register(c framework.Container) framework.NewInstance {
	if l.Driver == "" {
		tcs, err := c.Make(contract.ConfigKey)
		if err != nil {
			// 默认使用console
			return services.NewHadeConsoleLog
		}

		cs := tcs.(contract.Config)
		l.Driver = strings.ToLower(cs.GetString("log.Driver"))
	}

	// 根据driver的配置项确定
	switch l.Driver {
	case "single":
		return services.NewHadeSingleLog
	case "rotate":
		return services.NewHadeRotateLog
	case "console":
		return services.NewHadeConsoleLog
	case "custom":
		return services.NewHadeCustomLog
	default:
		return services.NewHadeConsoleLog
	}
}

// Boot 启动的时候注入
func (l *HadeLogServiceProvider) Boot(c framework.Container) error {
	return nil
}

// IsDefer 是否延迟加载
func (l *HadeLogServiceProvider) IsDefer() bool {
	return false
}

// Params 定义要传递给实例化方法的参数
func (l *HadeLogServiceProvider) Params(c framework.Container) []interface{} {
	// 获取configService
	configService := c.MustMake(contract.ConfigKey).(contract.Config)

	// 设置参数formatter
	if l.Formatter == nil {
		l.Formatter = formatter.TextFormatter
		if configService.IsExist("log.formatter") {
			v := configService.GetString("log.formatter")
			if v == "json" {
				l.Formatter = formatter.JsonFormatter
			} else if v == "text" {
				l.Formatter = formatter.TextFormatter
			}
		}
	}

	if l.Level == contract.UnknownLevel {
		l.Level = contract.InfoLevel
		if configService.IsExist("log.level") {
			l.Level = logLevel(configService.GetString("log.level"))
		}
	}

	// 定义5个参数
	return []interface{}{c, l.Level, l.CtxFielder, l.Formatter, l.Output}
}

// Name 定义对应的服务字符串凭证
func (l *HadeLogServiceProvider) Name() string {
	return contract.LogKey
}

// logLevel get level from string
func logLevel(config string) contract.LogLevel {
	switch strings.ToLower(config) {
	case "panic":
		return contract.PanicLevel
	case "fatal":
		return contract.FatalLevel
	case "error":
		return contract.ErrorLevel
	case "warn":
		return contract.WarnLevel
	case "info":
		return contract.InfoLevel
	case "debug":
		return contract.DebugLevel
	case "trace":
		return contract.TraceLevel
	}
	return contract.UnknownLevel
}
