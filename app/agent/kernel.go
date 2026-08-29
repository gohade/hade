package agent

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"

	"github.com/gohade/hade/app/agent/tool"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/pkg/errors"
)

// errAgentUnavailable 是 Agent 解析失败时唯一对外暴露的错误。
// 容器错误、Provider panic 等细节只进诊断输出，不进响应体。
var errAgentUnavailable = errors.New("agent service unavailable")

// NewAgentEngine 创建只承载 Agent API 的独立 HTTP 引擎。
//
// 构造期不做任何 Make/MustMake：非 Agent 命令（例如 hade build）也会走 main 的
// 引擎装配流程，不应该因此实例化 LLM/Agent Provider。
func NewAgentEngine(container framework.Container) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetContainer(container)
	engine.Use(gin.Recovery())

	Routes(engine)
	return engine, nil
}

// agentResolver 惰性解析容器中的 Agent，并保证示例工具只成功注册一次。
type agentResolver struct {
	mu         sync.Mutex
	agent      contract.Agent
	toolsReady bool
}

// resolve 返回缓存的 Agent；首次调用时从容器取实例并注册示例工具。
//
// 这里刻意不用 sync.Once：Once 一旦被 panic 的函数触发就永久标记完成，
// 示例工具会被永久丢掉。改成"先缓存 Agent，再用互斥锁下的布尔位记录注册成功"，
// 任一步失败都不留下半成品状态，后续请求可以重试。
func (r *agentResolver) resolve(c *gin.Context) (contract.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.agent == nil {
		agent, err := makeAgent(c)
		if err != nil {
			logDiagnostic("resolve agent: %v", err)
			return nil, errAgentUnavailable
		}
		r.agent = agent
	}
	if !r.toolsReady {
		if err := registerExampleToolsSafely(r.agent, c); err != nil {
			logDiagnostic("register example tools: %v", err)
			return nil, errAgentUnavailable
		}
		r.toolsReady = true
	}
	return r.agent, nil
}

// makeAgent 从容器解析 Agent 实例。
//
// Provider 的 Params 里普遍使用 MustMake 拉依赖（例如 HadeAgentProvider 会
// MustMake LLM），依赖缺失时会 panic。这里统一 recover 成 error，避免 panic
// 穿到 gin.Recovery 去生成一个空 body 的 500。
func makeAgent(c *gin.Context) (agent contract.Agent, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			agent = nil
			err = fmt.Errorf("container make panicked: %v\n%s", recovered, debug.Stack())
		}
	}()
	instance, err := c.Make(contract.AgentKey)
	if err != nil {
		return nil, err
	}
	typed, ok := instance.(contract.Agent)
	if !ok || typed == nil {
		return nil, errors.New("service " + contract.AgentKey + " is not a contract.Agent")
	}
	return typed, nil
}

// serviceLookup 用于判断是否绑定 ORM，以及把 MustMake 传给工具 Handler。
// *framework.HadeContainer 与 *gin.Context 均满足。
type serviceLookup interface {
	IsBind(key string) bool
	MustMake(key string) interface{}
}

// registerExampleToolsSafely 注册示例工具，并把第三方 RegisterTool 的 panic 转成 error。
func registerExampleToolsSafely(agent contract.Agent, lookup serviceLookup) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("register tool panicked: %v\n%s", recovered, debug.Stack())
		}
	}()
	RegisterExampleTools(agent, lookup)
	return nil
}

// logDiagnostic 把内部故障现场写到 stderr。
func logDiagnostic(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "[agent] "+format+"\n", args...)
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
