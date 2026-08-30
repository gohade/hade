# Agent HTTP 协议下沉到 framework（方案 A）

日期：2026-08-30  
状态：待用户确认后进入实现计划

## 背景与目标

`app/agent/handler.go` 承载的是 Agent **HTTP/SSE 协议**（升流时机、Run 并发与 recover、错误映射、请求体上限），不是业务。业务层应只负责：创建独立 Engine、挂路由、注册工具。

本设计把标准 Handler 下沉到 framework，**不改变对外 HTTP/SSE 契约**（路径、状态码、事件顺序与 JSON 字段）。

**成功标准：**

- `app/agent` 不再包含 SSE 循环、error→HTTP 映射、body limiter。
- 业务挂载方式为 `agenthttp.Mount(engine)`（或等价的三个 Handler）。
- 现有协议测试（含 panic 补 error、预流式 JSON 错误、body/message 上限）全部仍绿。
- `NewAgentEngine` 构造期仍不 `Make` LLM/Agent。

## 明确不做

- 不改 `contract.Agent` / ReAct / 事件类型。
- 不把 User/echo/time 工具放进 framework。
- 不在 `NewAgentEngine` 构造期实例化 Agent。
- 不把路径做成可配置（仍为 `/sessions`、`/sessions/:id`、`/sessions/:id/messages`）。
- 不提交密钥；不改 DeepSeek/User 工具行为。

## 包与职责

新建包 `framework/agenthttp`（不放进 vendored `framework/gin`，也不塞进 `framework/provider/agent`，避免「默认实现」和「HTTP 协议」缠在一起）。

| 路径 | 职责 |
|------|------|
| `framework/agenthttp/handler.go` | 现 `app/agent/handler.go` 的协议实现 |
| `framework/agenthttp/mount.go` | `Mount(*gin.Engine)` |
| `framework/agenthttp/*_test.go` | 原 engine_test / engine_json / engine_panic，以及 oversized body/message 等协议测试 |
| `app/agent/kernel.go` | `NewAgentEngine`：gin + Recovery + 工具惰性注册中间件 + `agenthttp.Mount` |
| `app/agent/route.go` | 删除或变成对 `Mount` 的一行转发 |
| `app/agent/handler.go` | 删除 |
| `app/agent/kernel_tools_test.go`、`app/agent/tool/*` | 不动 |

## 框架 API

```go
package agenthttp

func Mount(engine *gin.Engine)
func CreateSession(c *gin.Context)
func GetSession(c *gin.Context)
func Messages(c *gin.Context)
```

Handler 从 `c.Make(contract.AgentKey)` 取 Agent（与今天 resolve 的 Make 路径一致）。Make 失败或类型不对：HTTP 500 JSON，`error` 文案不泄露 panic 栈（栈只写 stderr，对齐现有 `errAgentUnavailable` 对外语义：业务不可用时对客户端仍是 500，不出现 `panic` 字样）。

**惰性工具注册仍在 app：** 框架 Handler 假定容器里已经能 Make 出 Agent。app 在 `Mount` 之前加中间件：首次请求 `Make` Agent 并 `RegisterExampleTools`，失败则 500 且可重试（保留现 `agentResolver` 语义）。中间件必须很薄，协议细节不放进去。

推荐形状：

```go
func NewAgentEngine(container framework.Container) (*gin.Engine, error) {
    gin.SetMode(gin.ReleaseMode)
    engine := gin.New()
    engine.SetContainer(container)
    engine.Use(gin.Recovery())
    engine.Use(newToolBootstrap().Middleware) // 惰性注册业务工具
    agenthttp.Mount(engine)
    return engine, nil
}
```

`Routes` 若只剩 `Mount`，可内联进 `NewAgentEngine` 并删除 `route.go`。

## 从 app 挪走的逻辑（行为保持）

1. **SSE `Messages`：** goroutine 跑 `Run`、`recover`→`ErrInternal`、`awaitFirstEvent`、预流式 JSON 错误、升流后补 error/done、`writeSSE` id 从 1 递增。
2. **`writeAgentError` 状态码映射**（400/404/409/413/503/500）与 SSE `error` 事件的 code/message（message 对非 internal 用 sentinel 名）。
3. **请求体上限：** 现实现为兼容 Go 1.18 的自写 `limitedBody`。仓库已是 **Go 1.25**，改为 `http.MaxBytesReader` + `http.MaxBytesError`（或等价 `errors.Is`），状态码仍 413。上限仍是 `contract.RequestBodyMaxBytes`。

channel 仍由 **HTTP 层在 `Run` 返回后关闭**，`Run` 不关 channel。

## 测试迁移

迁到 `framework/agenthttp`（用 `gin.New()` + `SetContainer` + `Mount`，绑定 stub Agent，**不**走 `NewAgentEngine`）：

- Session + SSE 帧顺序与 JSON
- 预流式错误
- snake_case wire
- Run panic 预流式 / 已升流
- 补 error 不重复
- 工具 panic 仍能到 final
- body / message 上限与 64KiB 可达

留在 `app/agent`：

- 构造 Engine 不实例化 Agent
- 未绑定 Agent 时构造成功、首请求 500
- 真实 Agent Provider 缺 LLM 时不在构造期 panic
- 工具注册 panic 可重试
- Session 上限 → 503（若该测试依赖 NewAgentEngine+resolver，可留 app；若只测 HTTP 映射，可放到 agenthttp）
- `RegisterExampleTools` 与 User 工具单测

`CreateSessionMapsSessionLimitTo503` 测的是错误映射，放到 `agenthttp` 更合适。

## 实现顺序

1. 新增 `framework/agenthttp`，搬 handler + 协议测试，确认 `go test ./framework/agenthttp` 绿。
2. `NewAgentEngine` 改为中间件 + `Mount`，删 `app/agent/handler.go`（及空的 `route.go`）。
3. `go test ./app/agent ./framework/agenthttp ./framework/provider/agent`。
