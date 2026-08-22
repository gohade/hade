package agent

import "github.com/gohade/hade/framework/gin"

// Routes 绑定 Agent Session API。
func Routes(engine *gin.Engine) {
	handler := &Handler{}
	engine.POST("/sessions", handler.CreateSession)
	engine.GET("/sessions/:id", handler.GetSession)
	engine.POST("/sessions/:id/messages", handler.Messages)
}
