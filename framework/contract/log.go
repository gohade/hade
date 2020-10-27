package contract

import (
	"context"
	"io"
	"time"
)

const LogKey = "hade:log"

type LogLevel uint32

const (
	// Unknownlevel is default level, which will design by provider
	UnknownLevel LogLevel = iota
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

var AllLevels = []LogLevel{
	PanicLevel,
	FatalLevel,
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
	TraceLevel,
}

// CtxFielder define ctx field which add to log field
type CtxFielder func(ctx context.Context) map[string]interface{}

// Formatter define fields format handler to string
type Formatter func(level LogLevel, t time.Time, msg string, fields map[string]interface{}) ([]byte, error)

// Log define interface for log
type Log interface {
	// Panic will call panic(fields) for debug
	Panic(ctx context.Context, msg string, fields map[string]interface{})
	// Fatal will add fatal record which contains msg and fields
	Fatal(ctx context.Context, msg string, fields map[string]interface{})
	// Error will add error record which contains msg and fields
	Error(ctx context.Context, msg string, fields map[string]interface{})
	// Warn will add warn record which contains msg and fields
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	// Info will add info record which contains msg and fields
	Info(ctx context.Context, msg string, fields map[string]interface{})
	// Debug will add debug record which contains msg and fields
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	// Trace will add trace info which contains msg and fields
	Trace(ctx context.Context, msg string, fields map[string]interface{})

	// SetLevel set log level, and higher level will be recorded
	SetLevel(level LogLevel)
	// SetCxtFielder will get fields from context
	SetCxtFielder(handler CtxFielder)
	// SetFormatter will set formatter handler will covert data to string for recording
	SetFormatter(formatter Formatter)
	// SetOutput will set output writer
	SetOutput(out io.Writer)
}

// FileLog define interface which fileLogger should satisfied
type SingleFileLog interface {
	Log
	SetFile(file string)
	SetFolder(folder string)
}

type RotatingFileLog interface {
	Log
	SetFolder(folder string)
	SetFile(file string)
	SetMaxFiles(maxFiles int)
	SetDateFormat(dateFormat string)
}

type ConsoleLog interface {
	Log
}
