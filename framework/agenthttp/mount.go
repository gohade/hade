package agenthttp

import "github.com/gohade/hade/framework/gin"

// Mount 绑定标准 Agent Session HTTP API。
func Mount(engine *gin.Engine) {
	MountWithConsole(engine, true)
}

// MountWithConsole 绑定 Agent API，并按开关决定是否暴露内置调试台及其配置接口。
// Session API 始终注册，保证关闭 Console 不影响业务 Agent 调用。
func MountWithConsole(engine *gin.Engine, consoleEnabled bool) {
	if consoleEnabled {
		engine.GET("/", ServeUI)
		engine.GET("/agent-ui.css", ServeUIAsset)
		engine.GET("/agent-ui.js", ServeUIAsset)
		engine.GET("/config", GetConfig)
		engine.PUT("/config", UpdateConfig)
		engine.GET("/tools", ListTools)
	}
	engine.POST("/sessions", CreateSession)
	engine.GET("/sessions/:id", GetSession)
	engine.POST("/sessions/:id/messages", Messages)
}
