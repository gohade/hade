# hade:trace

## 说明

hade:trace 是提供分布式链路追踪服务，可以用于跟踪分布式服务调用链路。

## 使用方法

```

// Trace define struct according Google Dapper
type TraceContext struct {
	TraceID  string // traceID global unique
	ParentID string // 父节点SpanID
	SpanID   string // 当前节点SpanID
	CspanID  string // 子节点调用的SpanID, 由调用方指定

	Annotation map[string]string // 标记各种信息
}

type Trace interface {
	// // SetTraceIDService set TraceID generator, default hadeIDGenerator
	// SetTraceIDService(IDService)
	// // SetTraceIDService set SpanID generator, default hadeIDGenerator
	// SetSpanIDService(IDService)

	// WithContext register new trace to context
	WithTrace(c context.Context, trace *TraceContext) context.Context
	// GetTrace From trace context
	GetTrace(c context.Context) *TraceContext
	// NewTrace generate a new trace
	NewTrace() *TraceContext
	// StartSpan generate cspan for child call
	StartSpan(trace *TraceContext) *TraceContext

	// traceContext to map for logger
	ToMap(trace *TraceContext) map[string]string

	// GetTrace By Http
	ExtractHTTP(req *http.Request) *TraceContext
	// Set Trace to Http
	InjectHTTP(req *http.Request, trace *TraceContext) *http.Request
}

```
