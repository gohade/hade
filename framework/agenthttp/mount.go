package agenthttp

import "github.com/gohade/hade/framework/gin"

// Mount 绑定标准 Agent Session HTTP API。
func Mount(engine *gin.Engine) {
	engine.POST("/sessions", CreateSession)
	engine.GET("/sessions/:id", GetSession)
	engine.POST("/sessions/:id/messages", Messages)
}
