package agent

import (
	"github.com/gohade/hade/app/agent/tool"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// NewAgentEngine 创建只承载 Agent API 的独立 HTTP 引擎。
func NewAgentEngine(container framework.Container) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetContainer(container)
	engine.Use(gin.Recovery())

	RegisterExampleTools(container)
	Routes(engine)
	return engine, nil
}

// RegisterExampleTools 注册无外部副作用的示例工具。
func RegisterExampleTools(container framework.Container) {
	agent := container.MustMake(contract.AgentKey).(contract.Agent)
	registered := make(map[string]struct{}, len(agent.ListTools()))
	for _, spec := range agent.ListTools() {
		registered[spec.Name] = struct{}{}
	}

	if _, ok := registered["echo"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "echo",
			Description: "返回输入文本",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{"type": "string"},
				},
			},
		}, tool.Echo)
	}
	if _, ok := registered["time"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "time",
			Description: "返回当前 UTC 时间",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}, tool.Time)
	}
}
