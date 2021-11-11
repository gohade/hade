package middleware

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// recovery机制，将协程中的函数异常进行捕获
func Trace() gin.HandlerFunc {
	// 使用函数回调
	return func(c *gin.Context) {
		// 记录开始时间
		tracer := c.MustMake(contract.TraceKey).(contract.Trace)
		traceCtx := tracer.ExtractHTTP(c.Request)

		tracer.WithTrace(c, traceCtx)

		// 使用next执行具体的业务逻辑
		c.Next()
	}
}
