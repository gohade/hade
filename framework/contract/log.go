package contract

import (
	"context"
	"io"
	"time"
)

const LogKey = "hade:log"

type LogLevel uint32

const (
	// UnknownLevel 表示未知的日志级别
	UnknownLevel LogLevel = iota
	// PanicLevel level, panic 表示会导致整个程序出现崩溃的日志信息
	PanicLevel
	// FatalLevel level. fatal 表示会导致当前这个请求出现提前终止的错误信息
	FatalLevel
	// ErrorLevel level. error 表示出现错误，但是不一定影响后续请求逻辑的错误信息
	ErrorLevel
	// WarnLevel level. warn 表示出现错误，但是一定不影响后续请求逻辑的报警信息
	WarnLevel
	// InfoLevel level. info 表示正常的日志信息输出
	InfoLevel
	// DebugLevel level. debug 表示在调试状态下打印出来的日志信息
	DebugLevel
	// TraceLevel level. trace 表示最详细的信息，一般信息量比较大，可能包含调用堆栈等信息
	TraceLevel
)

// CtxFielder 定义了从context中获取信息的方法
type CtxFielder func(ctx context.Context) map[string]interface{}

// Formatter 定义了将日志信息组织成字符串的通用方法
type Formatter func(level LogLevel, t time.Time, msg string, fields map[string]interface{}) ([]byte, error)

// Log 定义了日志服务协议
type Log interface {
	// Panic 表示会导致整个程序出现崩溃的日志信息
	Panic(ctx context.Context, msg string, fields map[string]interface{})
	// Fatal 表示会导致当前这个请求出现提前终止的错误信息
	Fatal(ctx context.Context, msg string, fields map[string]interface{})
	// Error 表示出现错误，但是不一定影响后续请求逻辑的错误信息
	Error(ctx context.Context, msg string, fields map[string]interface{})
	// Warn 表示出现错误，但是一定不影响后续请求逻辑的报警信息
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	// Info 表示正常的日志信息输出
	Info(ctx context.Context, msg string, fields map[string]interface{})
	// Debug 表示在调试状态下打印出来的日志信息
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	// Trace 表示最详细的信息，一般信息量比较大，可能包含调用堆栈等信息
	Trace(ctx context.Context, msg string, fields map[string]interface{})

	// SetLevel 设置日志级别
	SetLevel(level LogLevel)
	// SetCtxFielder 从context中获取上下文字段field
	SetCtxFielder(handler CtxFielder)
	// SetFormatter 设置输出格式
	SetFormatter(formatter Formatter)
	// SetOutput 设置输出管道
	SetOutput(out io.Writer)
}
