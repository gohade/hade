package agent

import (
	"github.com/gohade/hade/app/agent/tool"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/agenthttp"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// NewAgentEngine 创建只承载 Agent API 的独立 HTTP 引擎。
func NewAgentEngine(container framework.Container) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetContainer(container)
	engine.Use(gin.Recovery())
	registerExampleToolsFrom(container)
	agenthttp.Mount(engine)
	return engine, nil
}

func registerExampleToolsFrom(container framework.Container) {
	defer func() { _ = recover() }()
	if container == nil || !container.IsBind(contract.AgentKey) {
		return
	}
	instance, err := container.Make(contract.AgentKey)
	if err != nil {
		return
	}
	agent, ok := instance.(contract.Agent)
	if !ok || agent == nil {
		return
	}
	RegisterExampleTools(agent, container)
}

// serviceLookup 用于判断是否绑定 ORM，以及把 MustMake 传给工具 Handler。
// *framework.HadeContainer 与 *gin.Context 均满足。
type serviceLookup interface {
	IsBind(key string) bool
	MustMake(key string) interface{}
}

// RegisterExampleTools 注册示例工具。同名工具已存在时跳过。
// lookup 未绑定 hade:orm 时不注册 User 三工具。
func RegisterExampleTools(agent contract.Agent, lookup serviceLookup) {
	if agent == nil {
		return
	}
	registered := map[string]struct{}{}
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
	if lookup == nil || !lookup.IsBind(contract.ORMKey) {
		return
	}
	if _, ok := registered["create_user"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "create_user",
			Description: "在数据库中创建用户，返回新记录的 id 与字段",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]interface{}{
					"name":  map[string]interface{}{"type": "string"},
					"email": map[string]interface{}{"type": "string"},
					"age":   map[string]interface{}{"type": "integer"},
				},
			},
		}, tool.CreateUserHandler(lookup))
	}
	if _, ok := registered["get_user"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "get_user",
			Description: "按主键 id 查询一个用户",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer"},
				},
			},
		}, tool.GetUserHandler(lookup))
	}
	if _, ok := registered["list_users"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "list_users",
			Description: "列出用户，最多 20 条；可选按 name 模糊匹配",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
		}, tool.ListUsersHandler(lookup))
	}
}
