package demo2

import "github.com/gohade/hade/framework/gin"

// Demo2Middleware 代表中间件函数
func Demo2Middleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Next()
	}
}

