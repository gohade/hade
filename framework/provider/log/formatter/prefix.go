package formatter

import "github.com/gohade/hade/framework/contract"

func Prefix(level contract.LogLevel) string {
	prefix := ""
	switch level {
	case contract.PanicLevel:
		prefix = "[Panic]"
	case contract.FatalLevel:
		prefix = "[Fatal]"
	case contract.ErrorLevel:
		prefix = "[Error]"
	case contract.WarnLevel:
		prefix = "[Warn]"
	case contract.InfoLevel:
		prefix = "[Info]"
	case contract.DebugLevel:
		prefix = "[Debug]"
	case contract.TraceLevel:
		prefix = "[Trace]"
	}
	return prefix
}
