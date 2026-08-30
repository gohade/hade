# Agent HTTP 下沉 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Agent 的 HTTP/SSE 协议从 `app/agent/handler.go` 搬到 `framework/agenthttp`，业务层只装配 Engine 和注册工具。

**Architecture:** `agenthttp.Mount` 绑定 `/sessions` 三条路由；Handler 只 `Make(hade:agent)`。app 用薄中间件做惰性工具注册。对外契约不变。

**Tech Stack:** Go 1.25、`framework/gin`、`github.com/gin-contrib/sse`、`net/http.MaxBytesReader`、goconvey。

## Global Constraints

- 不改 `contract.Agent`、ReAct、SSE 事件类型/顺序/JSON 字段。
- 路径固定：`POST /sessions`、`GET /sessions/:id`、`POST /sessions/:id/messages`。
- 构造 `NewAgentEngine` 仍不 `Make` Agent/LLM。
- echo/time/User 工具留在 `app/agent`；`framework` 不得 import `app/`。
- 按用户要求：**实现过程中不要 git commit**，除非用户明确说提交。
- 不要改无关文件。

## File structure

| Path | Responsibility |
|------|----------------|
| `framework/agenthttp/handler.go` | CreateSession / GetSession / Messages、SSE、错误映射、body 限制 |
| `framework/agenthttp/mount.go` | `Mount(*gin.Engine)` |
| `framework/agenthttp/http_test.go` 等 | 从 app 迁来的协议测试 + 测试辅助 |
| `app/agent/kernel.go` | NewAgentEngine + 工具 bootstrap 中间件 |
| 删除 `app/agent/handler.go`、`app/agent/route.go` | 逻辑已搬走 |

---

### Task 1: `framework/agenthttp` 协议层（含测试）

**Files:**
- Create: `framework/agenthttp/mount.go`
- Create: `framework/agenthttp/handler.go`
- Create: `framework/agenthttp/http_test.go`（从 `app/agent/engine_test.go`、`engine_json_test.go`、`engine_panic_test.go` 以及 lazy 里的 oversized/503 测试迁入）

**Interfaces:**
- Consumes: `contract.Agent`、`contract.AgentKey`、`contract.RequestBodyMaxBytes`、`*gin.Engine` / `*gin.Context`。
- Produces:
  - `func Mount(engine *gin.Engine)`
  - `func CreateSession(c *gin.Context)`
  - `func GetSession(c *gin.Context)`
  - `func Messages(c *gin.Context)`

- [ ] **Step 1: 写测试辅助 + 一条最小失败测试**

`framework/agenthttp/http_test.go` 先放 helper 和 `TestMount_CreateSession201`。`newProtocolEngine` **不要**调用 `NewAgentEngine`：

```go
package agenthttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	"github.com/gohade/hade/framework/gin"
	. "github.com/smartystreets/goconvey/convey"
)

type agentStub struct {
	agent contract.Agent
}

func (stub *agentStub) Name() string { return contract.AgentKey }
func (stub *agentStub) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return stub.agent, nil }
}
func (stub *agentStub) Boot(framework.Container) error           { return nil }
func (stub *agentStub) IsDefer() bool                            { return false }
func (stub *agentStub) Params(framework.Container) []interface{} { return nil }

func newProtocolEngine(t *testing.T, agent contract.Agent) http.Handler {
	t.Helper()
	container := framework.NewHadeContainer()
	if err := container.Bind(&agentStub{agent: agent}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	engine.SetContainer(container)
	Mount(engine)
	return engine
}

func performRequest(handler http.Handler, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, body)
		request.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestMount_CreateSession201(t *testing.T) {
	Convey("Mount 后 POST /sessions 返回 201 和 id", t, func() {
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		engine := newProtocolEngine(t, mem)
		resp := performRequest(engine, http.MethodPost, "/sessions", nil)
		So(resp.Code, ShouldEqual, http.StatusCreated)
		So(resp.Body.String(), ShouldContainSubstring, `"id":`)
	})
}
```

需要 echo 的 SSE 测试（下一步搬文件时）在 **测试内** 注册本地 echo，禁止 `import app/agent/tool`：

```go
mem.RegisterTool(contract.ToolSpec{Name: "echo"}, func(_ context.Context, argsJSON string) (string, error) {
	var args struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(argsJSON), &args) == nil && args.Text != "" {
		return args.Text, nil
	}
	return argsJSON, nil
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./framework/agenthttp -count=1 -run TestMount_CreateSession201`

Expected: FAIL，`Mount` 未定义。

- [ ] **Step 3: 实现 Mount + 从 app 搬 handler**

`framework/agenthttp/mount.go`：

```go
package agenthttp

import "github.com/gohade/hade/framework/gin"

func Mount(engine *gin.Engine) {
	engine.POST("/sessions", CreateSession)
	engine.GET("/sessions/:id", GetSession)
	engine.POST("/sessions/:id/messages", Messages)
}
```

把 `app/agent/handler.go` **整文件**拷到 `framework/agenthttp/handler.go`，然后改：

1. `package agenthttp`
2. 删掉 `Handler` / `NewHandler` / `resolver`；三个导出函数直接 `Make` Agent：

```go
func agentFrom(c *gin.Context) (contract.Agent, error) {
	instance, err := c.Make(contract.AgentKey)
	if err != nil {
		return nil, err
	}
	typed, ok := instance.(contract.Agent)
	if !ok || typed == nil {
		return nil, errors.New("agent service unavailable")
	}
	return typed, nil
}
```

Make 失败时对客户端 JSON `{"error":"agent service unavailable"}`，500；stderr 可打内部 err。不要把 `contract.AgentKey` 或 panic 栈放进 body。

`CreateSession` / `GetSession` / `Messages` 开头改为 `agent, err := agentFrom(c)`，不再 `h.resolver.resolve`。

3. **body 限制：** 删 `limitedBody` / `errBodyTooLarge`。`Messages` 里：

```go
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, contract.RequestBodyMaxBytes)
```

`bindErrorStatus`：`errors.As` 到 `*http.MaxBytesError`（或 `errors.Is` 若包装后仍可判定）→ 413；否则 400。

4. 保留 `awaitFirstEvent`、`writeSSE`、`safeClose`、`errorEventData`、`writeAgentError` 行为与现文件一致（含 503 SessionLimit）。

- [ ] **Step 4: 把协议测试搬进 `framework/agenthttp`**

从 `app/agent` 拷贝并改 `package agenthttp`，`newTestEngine` 全部改为 `newProtocolEngine`：

- `engine_test.go` 里 Session+SSE、PreStreamErrors（含 stub 类型 `preStreamErrorAgent`）
- `engine_json_test.go`
- `engine_panic_test.go` 全部
- `TestMessagesRejectsOversizedBody`
- `TestMessagesRejectsOversizedMessage`
- `TestMessagesMessageLimitIsReachableUnderBodyLimit`
- `TestCreateSessionMapsSessionLimitTo503`

SSE 主测必须在创建 engine 前给 `MemoryAgent` 注册上面的本地 echo。

`app/agent` 里删掉已搬走的测试函数，**保留** `performRequest` / `assertJSONError` / `newTestEngine`（仍走 `NewAgentEngine`）供 lazy/工具测试使用。

- [ ] **Step 5: 跑协议测试**

Run: `go test ./framework/agenthttp -count=1`

Expected: PASS。

---

### Task 2: app 只保留装配与工具注册

**Files:**
- Modify: `app/agent/kernel.go`
- Delete: `app/agent/handler.go`
- Delete: `app/agent/route.go`
- Modify: `app/agent/engine_lazy_test.go`（去掉已迁走的 oversized/503；保留构造/重试/缺 LLM）

**Interfaces:**
- Consumes: `agenthttp.Mount`；现有 `RegisterExampleTools` / `agentResolver` 语义。
- Produces: `NewAgentEngine` 行为与现在一致（构造不 Make；首请求注册工具；失败 500 `agent service unavailable` 可重试）。

- [ ] **Step 1: 确认 app 侧失败点**

Run: `go test ./app/agent -count=1`

Expected: 可能 FAIL（`Routes`/`NewHandler` 已不存在，或重复测试）。按失败信息改。

- [ ] **Step 2: 改 `NewAgentEngine`**

```go
func NewAgentEngine(container framework.Container) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetContainer(container)
	engine.Use(gin.Recovery())
	engine.Use((&agentResolver{}).Middleware)
	agenthttp.Mount(engine)
	return engine, nil
}
```

把现有 `resolve` 改成中间件（先 resolve 再 `c.Next()`）。`resolve` 失败：写 500 JSON `agent service unavailable`（`c.Abort()`），与今天 `writeAgentError` 对 resolver 失败的表现一致。成功则 `c.Next()`，后续由 `agenthttp` 再 `Make` 一次——允许两次 Make（容器单例，第二次是缓存实例）。**不要**在中间件失败时把内部 Make 错误原文返回客户端。

删除 `Routes`、`NewHandler`、`app/agent/handler.go`、`app/agent/route.go`。

- [ ] **Step 3: 跑 app + agenthttp + provider/agent**

Run:

```
go test ./framework/agenthttp ./app/agent ./app/agent/tool ./framework/provider/agent -count=1
```

Expected: PASS。`TestNewAgentEngineDoesNotInstantiateAgent` 仍断言 echo/time 各一次；`ListTools` 长度 2。

- [ ] **Step 4: 不要 commit**（用户未要求提交。）

---

## Spec coverage

| Spec | Task |
|------|------|
| `framework/agenthttp` Mount/三 Handler | 1 |
| SSE/错误/body 从 app 搬走 | 1 |
| MaxBytesReader | 1 |
| 协议测试迁移 | 1 |
| NewAgentEngine 中间件 + Mount | 2 |
| 删 handler.go/route.go | 2 |
| 构造期不 Make；工具仍在 app | 2 |
