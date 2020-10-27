package log

import (
	"os"
	"strings"

	"hade/framework"
	"hade/framework/contract"
	"hade/framework/provider/log/formatter"
	"hade/framework/provider/log/services"
	"hade/framework/util"
)

type HadeLogServiceProvider struct {
	framework.ServiceProvider

	driver  string // driver
	configs map[string]interface{}

	// common config for log
	Formatter  contract.Formatter
	Level      contract.LogLevel
	CtxFielder contract.CtxFielder

	c framework.Container
}

// Register registe a new function for make a service instance
func (l *HadeLogServiceProvider) Register(c framework.Container) framework.NewInstance {
	tcs, err := c.Make(contract.ConfigKey)
	if err != nil {
		return services.NewHadeConsoleLog
	}

	cs := tcs.(contract.Config)

	l.driver = strings.ToLower(cs.GetString("log.driver"))
	l.configs = cs.GetStringMap("log")

	switch l.driver {
	case "single":
		return services.NewHadeSingleLog
	case "rotate":
		return services.NewHadeRotateLog
	case "console":
		return services.NewHadeConsoleLog
	default:
		return services.NewHadeConsoleLog
	}
}

// Boot will called when the service instantiate
func (l *HadeLogServiceProvider) Boot(c framework.Container) error {
	// Set Formatter/Level/CtxFielder
	if l.Formatter == nil {
		l.Formatter = formatter.TextFormatter
		if t, ok := l.configs["formatter"]; ok {
			v := t.(string)
			if v == "json" {
				l.Formatter = formatter.JsonFormatter
			} else if v == "text" {
				l.Formatter = formatter.TextFormatter
			}
		}
	}
	if l.Level == contract.UnknownLevel {
		if t, ok := l.configs["level"]; ok {
			l.Level = logLevel(t.(string))
		}
		if l.Level == contract.UnknownLevel {
			l.Level = contract.InfoLevel
		}
	}

	app := c.MustMake(contract.AppKey).(contract.App)

	switch l.driver {
	case "single":
		// check configs default: folder/file
		if _, ok := l.configs["folder"]; !ok {
			l.configs["folder"] = app.LogPath()
		}
		folder := l.configs["folder"].(string)
		if !util.Exists(folder) {
			os.MkdirAll(folder, os.ModePerm)
		}
		if _, ok := l.configs["file"]; !ok {
			l.configs["file"] = "hade.log"
		}
	case "rotate":
		// check configs default: folder/file
		if _, ok := l.configs["folder"]; !ok {
			l.configs["folder"] = app.LogPath()
		}
		folder := l.configs["folder"].(string)
		if !util.Exists(folder) {
			os.MkdirAll(folder, os.ModePerm)
		}
		if _, ok := l.configs["file"]; !ok {
			l.configs["file"] = "hade.log"
		}
		if _, ok := l.configs["max_files"]; !ok {
			l.configs["max_files"] = 30
		}
		if _, ok := l.configs["date_format"]; !ok {
			l.configs["date_format"] = "ymd"
		}
	}

	l.c = c
	return nil
}

// IsDefer define whether the service instantiate when first make or register
func (l *HadeLogServiceProvider) IsDefer() bool {
	return true
}

// Params define the necessary params for NewInstance
func (l *HadeLogServiceProvider) Params() []interface{} {
	// param sequence: level, ctxFielder, Formatter, map[string]string(folder/file)
	return []interface{}{l.Level, l.CtxFielder, l.Formatter, l.configs, l.c}
}

/// Name define the name for this service
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
