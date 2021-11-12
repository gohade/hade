# hade:log

提供日志记录相关操作

``` golang
type Log interface {
	// Panic will call panic(fields) for debug
	Panic(ctx context.Context, msg string, fields []interface{})
	// Fatal will add fatal record which contains msg and fields
	Fatal(ctx context.Context, msg string, fields []interface{})
	// Error will add error record which contains msg and fields
	Error(ctx context.Context, msg string, fields []interface{})
	// Warn will add warn record which contains msg and fields
	Warn(ctx context.Context, msg string, fields []interface{})
	// Info will add info record which contains msg and fields
	Info(ctx context.Context, msg string, fields []interface{})
	// Debug will add debug record which contains msg and fields
	Debug(ctx context.Context, msg string, fields []interface{})
	// Trace will add trace info which contains msg and fields
	Trace(ctx context.Context, msg string, fields []interface{})

	// SetLevel set log level, and higher level will be recorded
	SetLevel(level LogLevel)
	// SetCxtFielder will get fields from context
	SetCxtFielder(handler CtxFielder)
	// SetFormatter will set formatter handler will covert data to string for recording
	SetFormatter(formatter Formatter)
}

```