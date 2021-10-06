package mdemo

import "github.com/gohade/hade/framework/gin"

// MdemoMiddleware 代表中间件函数
func MdemoMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Next()
	}
}
