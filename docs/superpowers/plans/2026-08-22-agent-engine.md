# Agent Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 hade 增加与 HTTP/gRPC 平级的独立 Agent Engine：服务端 ReAct 内外循环、可替换 `hade:llm`、工具注册、SSE 事件流。

**Architecture:** Kernel 增加 `AgentEngine() http.Handler`；独立 gin 只挂 `/sessions` 路由，由 `hade agent start` 在 `agent.port`（默认 `:8889`）监听。ReAct 与 session 在 `hade:agent`，模型在 `hade:llm`。测试用 fake LLM，不打真实模型。

**Tech Stack:** Go 1.18、现有 `framework` 容器、vendored gin（`SSEvent` / `github.com/gin-contrib/sse`）、`net/http` OpenAI 兼容 Chat Completions、`github.com/google/uuid`、goconvey（与现有 provider 测试一致）。

## Global Constraints

- ReAct 循环全部在服务端；客户端只连 Agent 端口 SSE。
- 内循环 + 外循环（内存 session 历史）；同一 session `Run` 串行，冲突映射 HTTP 409。
- LLM 协议 Key 为 `hade:llm`；Agent 协议 Key 为 `hade:agent`。
- 第一版 LLM `Chat` 整段返回，不做 token 流；SSE 只推 Agent 事件。
- 示例工具仅 `echo`、`time`；禁止 shell / 读任意文件。
- `final` 与 `error` 互斥，之后都发 `done`；channel 由 Engine 在 `Run` 返回后关闭。
- thought/observation/final 以及 `GET /sessions` 的 content 截断 4096 字节；`api_key` 不得出现在响应中。
- 自动化测试禁止真实 LLM HTTP；用 fake Provider 与 JSON fixture。
- 提交信息使用中文；不要改无关文件。

## File structure

| Path | Responsibility |
|------|----------------|
| `framework/contract/llm.go` | `LLMKey`、`LLM`、`ChatRequest`/`ChatResponse` |
| `framework/contract/agent.go` | `AgentKey`、`Agent`、`Session`、`ToolSpec`、`AgentEvent`、错误变量 |
| `framework/provider/llm/fake.go` | 测试用脚本 LLM |
| `framework/provider/llm/openai.go` | OpenAI 兼容 HTTP 实现 |
| `framework/provider/llm/openai_codec.go` | 请求/响应 JSON 编解码（可单测） |
| `framework/provider/llm/provider.go` | `HadeLLMProvider` |
| `framework/provider/llm/openai_codec_test.go` | fixture 编解码测试 |
| `framework/provider/agent/service.go` | 内存 session、工具表、`Run` |
| `framework/provider/agent/provider.go` | `HadeAgentProvider` |
| `framework/provider/agent/service_test.go` | session/并发/`Run` 测试 |
| `app/agent/kernel.go` | `NewAgentEngine` |
| `app/agent/route.go` | 路由 |
| `app/agent/handler.go` | HTTP handler + SSE |
| `app/agent/tool/echo.go` `app/agent/tool/time.go` | 示例工具 |
| `app/agent/engine_test.go` | httptest SSE |
| `framework/command/agent/*.go` | start/stop/restart/state |
| `config/{development,testing,production}/agent.yaml` `llm.yaml` | 配置 |
| `framework/contract/kernel.go` 等 | 第三引擎接入 |

---

### Task 1: LLM 协议与 Fake LLM

**Files:**
- Create: `framework/contract/llm.go`
- Create: `framework/provider/llm/fake.go`
- Create: `framework/provider/llm/fake_test.go`

**Interfaces:**
- Consumes: 无
- Produces: `contract.LLMKey = "hade:llm"`；`contract.LLM.Chat(ctx, ChatRequest) (ChatResponse, error)`；`FinishStop`/`FinishToolCalls`/`FinishLength`；`provider/llm.ScriptLLM`

- [ ] **Step 1: Write the failing test**

Create `framework/provider/llm/fake_test.go`:

```go
package llm

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScriptLLM_ReturnsScriptedToolCallThenStop(t *testing.T) {
	Convey("scripted chat", t, func() {
		script := &ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{Role: "assistant", Content: "need echo"},
				ToolCalls: []contract.ToolCall{{
					ID: "call_1", Name: "echo", Arguments: `{"text":"hi"}`,
				}},
				Finish: contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "done"},
				Finish:  contract.FinishStop,
			},
		}}
		var svc contract.LLM = script
		r1, err := svc.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldBeNil)
		So(r1.Finish, ShouldEqual, contract.FinishToolCalls)
		So(r1.ToolCalls[0].Name, ShouldEqual, "echo")
		r2, err := svc.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldBeNil)
		So(r2.Finish, ShouldEqual, contract.FinishStop)
		So(r2.Message.Content, ShouldEqual, "done")
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./framework/provider/llm/ -run TestScriptLLM_ReturnsScriptedToolCallThenStop -count=1`

Expected: FAIL，包不存在或 `ScriptLLM` 未定义。

- [ ] **Step 3: Write contract + ScriptLLM**

`framework/contract/llm.go`:

```go
package contract

import "context"

const LLMKey = "hade:llm"

const (
	FinishStop      = "stop"
	FinishToolCalls = "tool_calls"
	FinishLength    = "length"
)

type Message struct {
	Role       string // system | user | assistant | tool
	Content    string
	ToolCallID string // role=tool 时对应的 call id
	ToolCalls  []ToolCall
}

type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema object
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON object as string
}

type ChatRequest struct {
	Messages    []Message
	Tools       []ToolSpec
	MaxTokens   int
	Temperature float32
}

type ChatResponse struct {
	Message   Message
	ToolCalls []ToolCall
	Finish    string
}

type LLM interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
```

`framework/provider/llm/fake.go`:

```go
package llm

import (
	"context"
	"sync"

	"github.com/gohade/hade/framework/contract"
)

type ScriptLLM struct {
	mu        sync.Mutex
	Responses []contract.ChatResponse
	Errs      []error
	Calls     []contract.ChatRequest
	idx       int
}

func (s *ScriptLLM) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.ChatResponse{}, err
	}
	s.Calls = append(s.Calls, req)
	i := s.idx
	s.idx++
	if i < len(s.Errs) && s.Errs[i] != nil {
		return contract.ChatResponse{}, s.Errs[i]
	}
	if i >= len(s.Responses) {
		return contract.ChatResponse{}, contract.ErrLLMFailed
	}
	return s.Responses[i], nil
}
```

`ErrLLMFailed` 将在 Task 2 的 `contract/agent.go` 定义。为让 Task 1 能编译，先在 `framework/contract/llm.go` 增加：

```go
import "github.com/pkg/errors"

var ErrLLMFailed = errors.New("llm_failed")
```

Task 2 把该变量移到 `agent.go` 并在 `llm.go` 删除，改为 fake 返回 `fmt.Errorf("llm exhausted")` 即可避免循环依赖。实现时 **不要** 让 `llm` 包 import agent：`ScriptLLM` 脚本用尽时返回 `errors.New("llm script exhausted")`。

修正 `fake.go` 最后一行为：

```go
	return contract.ChatResponse{}, errors.New("llm script exhausted")
```

并 `import "github.com/pkg/errors"`。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./framework/provider/llm/ -run TestScriptLLM_ReturnsScriptedToolCallThenStop -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add framework/contract/llm.go framework/provider/llm/fake.go framework/provider/llm/fake_test.go
git commit -m "$(cat <<'EOF'
增加 hade:llm 协议与可脚本化的 Fake LLM。

EOF
)"
```

---

### Task 2: Agent 协议、内存 Session、工具注册

**Files:**
- Create: `framework/contract/agent.go`
- Create: `framework/provider/agent/provider.go`
- Create: `framework/provider/agent/service.go`
- Create: `framework/provider/agent/service_session_test.go`

**Interfaces:**
- Consumes: `github.com/google/uuid`；容器 `framework.Container`（可暂不使用）
- Produces: `contract.AgentKey = "hade:agent"`；`Agent` 接口；`ErrSessionNotFound`、`ErrSessionBusy`、`ErrEmptyMessage`、`ErrMaxIterations`、`ErrCanceled`；`RegisterTool`；`CreateSession`/`GetSession`

- [ ] **Step 1: Write the failing test**

`framework/provider/agent/service_session_test.go`:

```go
package agent

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryAgent_CreateAndGetSession(t *testing.T) {
	Convey("session crud", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		So(id, ShouldNotBeEmpty)
		s, err := a.GetSession(context.Background(), id)
		So(err, ShouldBeNil)
		So(s.ID, ShouldEqual, id)
		So(len(s.Messages), ShouldEqual, 0)
		_, err = a.GetSession(context.Background(), "missing")
		So(err, ShouldEqual, contract.ErrSessionNotFound)
	})
}

func TestMemoryAgent_RegisterEchoTool(t *testing.T) {
	Convey("tools", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		a.RegisterTool(contract.ToolSpec{
			Name:        "echo",
			Description: "echo text",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{"type": "string"},
				},
			},
		}, func(ctx context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})
		So(len(a.ListTools()), ShouldEqual, 1)
		So(a.ListTools()[0].Name, ShouldEqual, "echo")
	})
}

type fakeLLM struct{}

func (f *fakeLLM) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	return contract.ChatResponse{Finish: contract.FinishStop}, nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./framework/provider/agent/ -count=1`

Expected: FAIL，`NewMemoryAgent` 未定义。

- [ ] **Step 3: Write contract + memory agent (尚不实现 Run 循环细节，但签名要有)**

`framework/contract/agent.go`:

```go
package contract

import (
	"context"

	"github.com/pkg/errors"
)

const AgentKey = "hade:agent"

const (
	EventSession      = "session"
	EventThought      = "thought"
	EventAction       = "action"
	EventObservation  = "observation"
	EventFinal        = "final"
	EventError        = "error"
	EventDone         = "done"
	ContentMaxBytes   = 4096
	DefaultMaxIter    = 8
)

var (
	ErrSessionNotFound = errors.New("session_not_found")
	ErrSessionBusy     = errors.New("session_busy")
	ErrEmptyMessage    = errors.New("empty_message")
	ErrMaxIterations   = errors.New("max_iterations")
	ErrCanceled        = errors.New("canceled")
	ErrLLMFailed       = errors.New("llm_failed")
)

type Session struct {
	ID       string
	Messages []Message
}

type ToolHandler func(ctx context.Context, argsJSON string) (observation string, err error)

type AgentEvent struct {
	Type string
	Data map[string]interface{}
}

type Agent interface {
	CreateSession(ctx context.Context) (sessionID string, error)
	GetSession(ctx context.Context, id string) (Session, error)
	RegisterTool(spec ToolSpec, handler ToolHandler)
	ListTools() []ToolSpec
	Run(ctx context.Context, sessionID, userMessage string, events chan<- AgentEvent) error
}
```

若 Task 1 已在 `llm.go` 声明 `ErrLLMFailed`，只保留 **一处**：放在 `agent.go`，从 `llm.go` 删除。

`framework/provider/agent/service.go`（含互斥、Get/Create/Register，`Run` 先返回 `errors.New("not implemented")`）：

```go
package agent

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

type memorySession struct {
	data SessionData
	mu   sync.Mutex
}

type SessionData struct {
	ID       string
	Messages []contract.Message
}

type registeredTool struct {
	spec    contract.ToolSpec
	handler contract.ToolHandler
}

type MemoryAgent struct {
	llm     contract.LLM
	maxIter int
	mu      sync.RWMutex
	sess    map[string]*memorySession
	tools   []registeredTool
}

func NewMemoryAgent(llm contract.LLM, maxIter int) *MemoryAgent {
	if maxIter <= 0 {
		maxIter = contract.DefaultMaxIter
	}
	return &MemoryAgent{
		llm:     llm,
		maxIter: maxIter,
		sess:    map[string]*memorySession{},
	}
}

func NewHadeAgentService(params ...interface{}) (interface{}, error) {
	llm := params[0].(contract.LLM)
	maxIter := params[1].(int)
	return NewMemoryAgent(llm, maxIter), nil
}

func (a *MemoryAgent) CreateSession(ctx context.Context) (string, error) {
	id := uuid.New().String()
	a.mu.Lock()
	a.sess[id] = &memorySession{data: SessionData{ID: id}}
	a.mu.Unlock()
	return id, nil
}

func (a *MemoryAgent) GetSession(ctx context.Context, id string) (contract.Session, error) {
	a.mu.RLock()
	s, ok := a.sess[id]
	a.mu.RUnlock()
	if !ok {
		return contract.Session{}, contract.ErrSessionNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := append([]contract.Message(nil), s.data.Messages...)
	for i := range msgs {
		msgs[i].Content = truncate(msgs[i].Content, contract.ContentMaxBytes)
	}
	return contract.Session{ID: s.data.ID, Messages: msgs}, nil
}

func (a *MemoryAgent) RegisterTool(spec contract.ToolSpec, handler contract.ToolHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = append(a.tools, registeredTool{spec: spec, handler: handler})
}

func (a *MemoryAgent) ListTools() []contract.ToolSpec {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]contract.ToolSpec, len(a.tools))
	for i, t := range a.tools {
		out[i] = t.spec
	}
	return out
}

func (a *MemoryAgent) Run(ctx context.Context, sessionID, userMessage string, events chan<- contract.AgentEvent) error {
	return errors.New("not implemented")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
```

`framework/provider/agent/provider.go`：`IsDefer` 返回 true（需要先有 llm）。`Params`：`c.MustMake(contract.LLMKey).(contract.LLM)` 与 `maxIter`。`maxIter` 若 config 尚未绑定则用 `contract.DefaultMaxIter`：

```go
package agent

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeAgentProvider struct{}

func (p *HadeAgentProvider) Name() string { return contract.AgentKey }
func (p *HadeAgentProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeAgentService
}
func (p *HadeAgentProvider) Boot(c framework.Container) error { return nil }
func (p *HadeAgentProvider) IsDefer() bool                   { return true }
func (p *HadeAgentProvider) Params(c framework.Container) []interface{} {
	llm := c.MustMake(contract.LLMKey).(contract.LLM)
	maxIter := contract.DefaultMaxIter
	if c.IsBind(contract.ConfigKey) {
		cfg := c.MustMake(contract.ConfigKey).(contract.Config)
		if cfg.IsExist("agent.max_iterations") {
			maxIter = cfg.GetInt("agent.max_iterations")
		}
	}
	return []interface{}{llm, maxIter}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./framework/provider/agent/ -run 'TestMemoryAgent_CreateAndGetSession|TestMemoryAgent_RegisterEchoTool' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add framework/contract/agent.go framework/contract/llm.go framework/provider/agent/
git commit -m "$(cat <<'EOF'
增加 hade:agent 协议与内存 Session、工具注册。

EOF
)"
```

---

### Task 3: ReAct `Run` 内循环

**Files:**
- Modify: `framework/provider/agent/service.go`（实现 `Run`）
- Create: `framework/provider/agent/service_run_test.go`

**Interfaces:**
- Consumes: `contract.LLM.Chat`；`contract.ToolHandler`；`MemoryAgent.maxIter`
- Produces: 向 `events` 发送 `session`/`thought`/`action`/`observation`/`final`/`error`（**不发送 `done`**，由 HTTP 层发）；`Run` 返回对应 error；user 消息在 LLM 失败时仍保留

- [ ] **Step 1: Write the failing tests**

`framework/provider/agent/service_run_test.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"testing"

	llmp "github.com/gohade/hade/framework/provider/llm"
	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func collect(ch <-chan contract.AgentEvent) []contract.AgentEvent {
	var out []contract.AgentEvent
	for e := range ch {
		out = append(out, e)
	}
	return out
}

func TestRun_EchoReActThenFinal(t *testing.T) {
	Convey("react echo", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message:   contract.Message{Role: "assistant", Content: "call echo"},
				ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				Finish:    contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "ok"},
				Finish:  contract.FinishStop,
			},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 16)
		go func() {
			_ = a.Run(context.Background(), id, "hello", ch)
			close(ch)
		}()
		evs := collect(ch)
		types := make([]string, len(evs))
		for i, e := range evs {
			types[i] = e.Type
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		sess, _ := a.GetSession(context.Background(), id)
		So(len(sess.Messages) >= 2, ShouldBeTrue)
		So(sess.Messages[0].Role, ShouldEqual, "user")
		So(sess.Messages[0].Content, ShouldEqual, "hello")
	})
}

func TestRun_MaxIterations(t *testing.T) {
	Convey("max iter", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
			{ToolCalls: []contract.ToolCall{{ID: "c2", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
			{ToolCalls: []contract.ToolCall{{ID: "c3", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
		}}
		a := NewMemoryAgent(script, 2)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			return "x", nil
		})
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 32)
		var runErr error
		go func() {
			runErr = a.Run(context.Background(), id, "go", ch)
			close(ch)
		}()
		evs := collect(ch)
		So(runErr, ShouldEqual, contract.ErrMaxIterations)
		last := evs[len(evs)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "max_iterations")
		for _, e := range evs {
			So(e.Type, ShouldNotEqual, contract.EventFinal)
		}
	})
}

func TestRun_SessionBusy(t *testing.T) {
	Convey("busy", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		a.mu.RLock()
		s := a.sess[id]
		a.mu.RUnlock()
		s.mu.Lock()
		ch := make(chan contract.AgentEvent, 2)
		err := a.Run(context.Background(), id, "two", ch)
		s.mu.Unlock()
		So(err, ShouldEqual, contract.ErrSessionBusy)
	})
}

func TestRun_KeepUserOnLLMError(t *testing.T) {
	Convey("keep user", t, func() {
		script := &llmp.ScriptLLM{Errs: []error{contract.ErrLLMFailed}}
		a := NewMemoryAgent(script, 8)
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 8)
		go func() {
			_ = a.Run(context.Background(), id, "keepme", ch)
			close(ch)
		}()
		collect(ch)
		sess, _ := a.GetSession(context.Background(), id)
		So(sess.Messages[0].Content, ShouldEqual, "keepme")
	})
}
```

`Run` 对 session 使用 `TryLock()`（Go 1.18 `sync.Mutex.TryLock`）。拿不到锁时返回 `contract.ErrSessionBusy`，不要阻塞等待。

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./framework/provider/agent/ -run 'TestRun_' -count=1`

Expected: FAIL，`Run` 仍是 not implemented。

- [ ] **Step 3: Implement Run**

替换 `MemoryAgent.Run`：

```go
func (a *MemoryAgent) Run(ctx context.Context, sessionID, userMessage string, events chan<- contract.AgentEvent) error {
	if strings.TrimSpace(userMessage) == "" {
		return contract.ErrEmptyMessage
	}
	a.mu.RLock()
	s, ok := a.sess[sessionID]
	a.mu.RUnlock()
	if !ok {
		return contract.ErrSessionNotFound
	}
	if !s.mu.TryLock() {
		return contract.ErrSessionBusy
	}
	defer s.mu.Unlock()

	send := func(typ string, data map[string]interface{}) {
		if events != nil {
			events <- contract.AgentEvent{Type: typ, Data: data}
		}
	}
	send(contract.EventSession, map[string]interface{}{"session_id": sessionID})

	s.data.Messages = append(s.data.Messages, contract.Message{Role: "user", Content: userMessage})

	emitTrunc := func(typ, key, val string) {
		send(typ, map[string]interface{}{key: truncate(val, contract.ContentMaxBytes)})
	}

	for i := 0; i < a.maxIter; i++ {
		if err := ctx.Err(); err != nil {
			send(contract.EventError, map[string]interface{}{"code": "canceled", "message": err.Error()})
			return contract.ErrCanceled
		}
		tools := a.ListTools()
		resp, err := a.llm.Chat(ctx, contract.ChatRequest{Messages: append([]contract.Message(nil), s.data.Messages...), Tools: tools})
		if err != nil {
			send(contract.EventError, map[string]interface{}{"code": "llm_failed", "message": err.Error()})
			return contract.ErrLLMFailed
		}
		emitTrunc(contract.EventThought, "content", resp.Message.Content)

		if len(resp.ToolCalls) > 0 {
			asst := contract.Message{Role: "assistant", Content: resp.Message.Content, ToolCalls: resp.ToolCalls}
			s.data.Messages = append(s.data.Messages, asst)
			for _, tc := range resp.ToolCalls {
				argsMap := map[string]interface{}{}
				raw := tc.Arguments
				if json.Unmarshal([]byte(tc.Arguments), &argsMap) != nil {
					argsMap = map[string]interface{}{}
				}
				send(contract.EventAction, map[string]interface{}{"name": tc.Name, "arguments": argsMap})
				obs, herr := a.execTool(ctx, tc.Name, raw)
				if herr != nil {
					obs = herr.Error()
				}
				obs = truncate(obs, contract.ContentMaxBytes)
				if json.Unmarshal([]byte(tc.Arguments), &map[string]interface{}{}) != nil {
					obs = truncate("invalid tool arguments: "+tc.Arguments+" ; "+obs, contract.ContentMaxBytes)
				}
				send(contract.EventObservation, map[string]interface{}{"name": tc.Name, "content": obs})
				s.data.Messages = append(s.data.Messages, contract.Message{
					Role: "tool", Content: obs, ToolCallID: tc.ID,
				})
			}
			continue
		}
		s.data.Messages = append(s.data.Messages, contract.Message{Role: "assistant", Content: resp.Message.Content})
		emitTrunc(contract.EventFinal, "content", resp.Message.Content)
		return nil
	}
	send(contract.EventError, map[string]interface{}{"code": "max_iterations", "message": "max iterations reached"})
	return contract.ErrMaxIterations
}

func (a *MemoryAgent) execTool(ctx context.Context, name, argsJSON string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, t := range a.tools {
		if t.spec.Name == name {
			return t.handler(ctx, argsJSON)
		}
	}
	return "", errors.New("unknown tool: " + name)
}
```

补上 `import`：`encoding/json`、`strings`。

非法 JSON 检测不要写两次 Unmarshal；抽成 `valid := json.Unmarshal(...) == nil`。

- [ ] **Step 4: Run tests**

Run: `go test ./framework/provider/agent/ -count=1`

Expected: PASS（含 Task 2 测试）

- [ ] **Step 5: Commit**

```bash
git add framework/provider/agent/
git commit -m "$(cat <<'EOF'
实现服务端 ReAct 循环与 Session 串行锁。

EOF
)"
```

---

### Task 4: AgentEngine HTTP + SSE

**Files:**
- Create: `app/agent/kernel.go`
- Create: `app/agent/route.go`
- Create: `app/agent/handler.go`
- Create: `app/agent/tool/echo.go`
- Create: `app/agent/tool/time.go`
- Create: `app/agent/engine_test.go`

**Interfaces:**
- Consumes: `contract.Agent`（从 gin container `MustMake(AgentKey)`）；`contract.AgentEvent`
- Produces: `agent.NewAgentEngine(container) (*gin.Engine, error)`；`POST /sessions`；`GET /sessions/:id`；`POST /sessions/:id/messages` SSE；事件末尾由 handler 写 `done` 并 `close` 不由 Run close（测试里与 Task 3 一样由调用方 close；handler 在 `Run` 返回后发 `done` 再 return，**不要 close 传给 Run 的 channel**，用缓冲 channel + `Run` 同步调用）

Handler 中 **同步** `Run`，channel 缓冲 64：

```go
events := make(chan contract.AgentEvent, 64)
done := make(chan error, 1)
go func() {
	done <- ag.Run(c.Request.Context(), id, req.Message, events)
	close(events)
}()
seq := 0
c.Header("Content-Type", "text/event-stream")
c.Header("Cache-Control", "no-cache")
c.Header("Connection", "keep-alive")
c.Status(200)
for ev := range events {
	seq++
	c.Render(-1, sse.Event{Event: ev.Type, Id: strconv.Itoa(seq), Data: ev.Data})
	c.Writer.Flush()
}
err := <-done
seq++
if err != nil {
	// Run 已发 error 事件；仍发 done
}
c.Render(-1, sse.Event{Event: contract.EventDone, Id: strconv.Itoa(seq), Data: map[string]interface{}{}})
c.Writer.Flush()
```

未升流错误：`ErrSessionNotFound` → 404；`ErrSessionBusy` → 409；`ErrEmptyMessage` → 400。JSON `{"error":"..."}`。

- [ ] **Step 1: Write httptest**

`app/agent/engine_test.go`：绑定 MemoryAgent + ScriptLLM + echo，`httptest` `POST /sessions` 再 `POST /sessions/:id/messages`，body 含 `event:thought` 与 `event:done`。

```go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMessagesSSE_EchoLoop(t *testing.T) {
	Convey("sse", t, func() {
		c := framework.NewHadeContainer()
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message:   contract.Message{Content: "t1"},
				ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				Finish:    contract.FinishToolCalls,
			},
			{Message: contract.Message{Content: "bye"}, Finish: contract.FinishStop},
		}}
		mem := agprovider.NewMemoryAgent(script, 8)
		mem.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})
		_ = c.Bind(&agentStub{a: mem})
		engine, err := NewAgentEngine(c)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sessions", nil)
		engine.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 201)
		var created struct{ ID string `json:"id"` }
		_ = json.Unmarshal(w.Body.Bytes(), &created)
		body := bytes.NewBufferString(`{"message":"hello"}`)
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPost, "/sessions/"+created.ID+"/messages", body)
		req2.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(w2, req2)
		out := w2.Body.String()
		So(strings.Contains(out, "event:session"), ShouldBeTrue)
		So(strings.Contains(out, "event:thought"), ShouldBeTrue)
		So(strings.Contains(out, "event:action"), ShouldBeTrue)
		So(strings.Contains(out, "event:observation"), ShouldBeTrue)
		So(strings.Contains(out, "event:final"), ShouldBeTrue)
		So(strings.Contains(out, "event:done"), ShouldBeTrue)
	})
}

type agentStub struct {
	a contract.Agent
}

func (s *agentStub) Name() string { return contract.AgentKey }
func (s *agentStub) Register(c framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return s.a, nil }
}
func (s *agentStub) Boot(c framework.Container) error { return nil }
func (s *agentStub) IsDefer() bool                    { return false }
func (s *agentStub) Params(c framework.Container) []interface{} {
	return nil
}
```

再加两个用例：`POST /sessions/nope/messages` 且 body `{"message":"x"}` 期望 404；`POST /sessions/{id}/messages` body `{"message":""}` 期望 400。

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/agent/ -count=1`

Expected: FAIL，`NewAgentEngine` 未定义。

- [ ] **Step 3: Implement engine**

`app/agent/tool/echo.go`：解析 `{"text":"..."}`，无 text 则返回整个 argsJSON。

`app/agent/tool/time.go`：`time.Now().UTC().Format(time.RFC3339)`。

`app/agent/kernel.go`：`gin.New()` + Recovery + `SetContainer` + `RegisterExampleTools` + `Routes`。

`RegisterExampleTools`：从 container 取 Agent，注册 echo/time。

`handler.go`：如上 SSE；`GET` 返回 `GetSession` JSON。

- [ ] **Step 4: Run tests**

Run: `go test ./app/agent/ ./framework/provider/agent/ -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/agent/
git commit -m "$(cat <<'EOF'
增加独立 AgentEngine 的 Session 与 SSE 接口。

EOF
)"
```

---

### Task 5: OpenAI 兼容 Provider（仅 fixture，无真网）

**Files:**
- Create: `framework/provider/llm/openai_codec.go`
- Create: `framework/provider/llm/openai.go`
- Create: `framework/provider/llm/provider.go`
- Create: `framework/provider/llm/openai_codec_test.go`

**Interfaces:**
- Consumes: `contract.ChatRequest` / `ChatResponse`；config `llm.base_url` `llm.api_key` `llm.model`
- Produces: `HadeLLMProvider`；`BuildOpenAIRequest` / `ParseOpenAIResponse`

- [ ] **Step 1: Write codec test**

`openai_codec_test.go` 用这段 fixture（不要发起 HTTP）：

请求：一条 user + 一个 echo tool。断言 JSON 含 `"role":"user"` 且 `tools[0].function.name == "echo"`。

响应 fixture：

```json
{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"echo","arguments":"{\"text\":\"hi\"}"}}]}}]}
```

断言 `Finish == FinishToolCalls` 且 `ToolCalls[0].Arguments` 为 `{"text":"hi"}`。

第二条 fixture `finish_reason: stop` + `content: "bye"`。

- [ ] **Step 2: Run to fail**

Run: `go test ./framework/provider/llm/ -run Codec -count=1`

Expected: FAIL

- [ ] **Step 3: Implement codec + HTTP client**

`openai.go`：`POST {base_url}/chat/completions`（`base_url` 若已含 `/v1` 则不要重复）。Header `Authorization: Bearer {api_key}`。`api_key` 为空则 `Chat` 返回 `contract.ErrLLMFailed`。`http.Client` 超时 60s。非 2xx 返回 `ErrLLMFailed`。

`provider.go`：`IsDefer true`；`Params` 读 config，缺省 `https://api.openai.com/v1`、`gpt-4o-mini`、空 key。

- [ ] **Step 4: Run tests**

Run: `go test ./framework/provider/llm/ -count=1`

Expected: PASS，无外网。

- [ ] **Step 5: Commit**

```bash
git add framework/provider/llm/
git commit -m "$(cat <<'EOF'
增加可替换的 OpenAI 兼容 LLM Provider。

EOF
)"
```

---

### Task 6: Kernel、main、命令、配置

**Files:**
- Modify: `framework/contract/kernel.go`
- Modify: `framework/provider/kernel/service.go`
- Modify: `framework/provider/kernel/provider.go`
- Modify: `main.go`
- Modify: `framework/command/kernel.go`
- Create: `framework/command/agent/agent.go`
- Create: `framework/command/agent/start.go`
- Create: `framework/command/agent/stop.go`
- Create: `framework/command/agent/restart.go`
- Create: `framework/command/agent/state.go`
- Create: `config/development/agent.yaml`、`config/development/llm.yaml`
- Create: 同样文件于 `config/testing/`、`config/production/`

**Interfaces:**
- Consumes: `kernel.AgentEngine() http.Handler`；`config agent.port` 默认 `:8889`
- Produces: `hade agent start|stop|restart|state`；pid `{runtime}/agent.pid`

- [ ] **Step 1: Write a compile-level test for Kernel interface**

在 `framework/provider/kernel/service_test.go`：

```go
package kernel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohade/hade/framework/gin"
	"google.golang.org/grpc"
)

func TestHadeKernelService_AgentEngine(t *testing.T) {
	e := gin.New()
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	s, err := NewHadeKernelService(e, grpc.NewServer(), e)
	if err != nil {
		t.Fatal(err)
	}
	ks := s.(*HadeKernelService)
	w := httptest.NewRecorder()
	ks.AgentEngine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Body.String() != "pong" {
		t.Fatalf("got %q", w.Body.String())
	}
}
```

- [ ] **Step 2: Run to fail**

Run: `go test ./framework/provider/kernel/ -count=1`

Expected: FAIL，`NewHadeKernelService` 参数个数不对。

- [ ] **Step 3: Wire kernel + main + commands + yaml**

`Kernel` 增加 `AgentEngine() http.Handler`。

`HadeKernelProvider` 增加字段 `AgentEngine *gin.Engine`；`Boot` 若 nil 则 `gin.Default()` 并 `SetContainer`；`Params` 三个 engine。

`NewHadeKernelService` 第三参数 `*gin.Engine`。

`main.go`：

```go
_ = container.Bind(&llm.HadeLLMProvider{})
_ = container.Bind(&agentprovider.HadeAgentProvider{})
// after grpc:
if engine, err := agentapp.NewAgentEngine(container); err == nil {
    kernelProvider.AgentEngine = engine
}
```

注意 import 别名避免 `agent` 冲突。

`framework/command/agent/start.go`：从 `framework/command/app.go` 复制 `startAppServe` 与 `appStartCommand` 的 RunE，改动如下：

- `core := kernelService.AgentEngine()`
- 地址：flag `--address` → env `AGENT_ADDRESS` → `config.IsExist("agent.port")` 则拼 `":" + port` 若 port 不含冒号，否则用 `agent.port` 原样；默认 `:8889`
- pid：`agent.pid`；log：`agent.log`；进程名 `hade agent`
- daemon `Use`/`Short` 文案改为 agent

`stop.go` / `state.go`：复制 `framework/command/grpc/stop.go` 与 `state.go`，pid 文件改为 `agent.pid`，打印「agent服务」。

`restart.go`：复制 `framework/command/grpc/restart.go`，将 `grpc.pid` 改为 `agent.pid`，将 `grpcStartCommand` 改为 `agentStartCommand`，文案改为 agent。

`agent.go`：`InitAgentCommand` 与 `InitGrpcCommand` 相同结构，`Use: "agent"`。

`AddKernelCommands`：`root.AddCommand(agentcmd.InitAgentCommand())`。

yaml：

`config/development/agent.yaml`：

```yaml
port: ":8889"
max_iterations: 8
```

`config/development/llm.yaml`：

```yaml
base_url: "https://api.openai.com/v1"
api_key: ""
model: "gpt-4o-mini"
```

testing/production 同样内容（production 的 api_key 保持空字符串占位）。

- [ ] **Step 4: Run tests + build**

Run:

```
go test ./framework/provider/kernel/ ./framework/provider/agent/ ./framework/provider/llm/ ./app/agent/ -count=1
go build -o /tmp/hade-agent-check .
```

Expected: 测试 PASS；`go build` 成功。`go run . agent` 能打印 help（可选：`go run . agent -h`）。

- [ ] **Step 5: Commit**

```bash
git add framework/contract/kernel.go framework/provider/kernel/ main.go framework/command/kernel.go framework/command/agent/ config/ app/agent/
git commit -m "$(cat <<'EOF'
接入 Kernel 第三引擎与 hade agent 启停命令。

EOF
)"
```

---

## Self-review

**Spec coverage:**

| Spec 项 | Task |
|---------|------|
| 独立 AgentEngine / Kernel 三足 / `hade agent start` | 6 |
| `hade:llm` + OpenAI 兼容 + 可替换 | 1, 5 |
| `hade:agent` session 内外循环、内存存储 | 2, 3 |
| 工具注册 + echo/time | 2, 4 |
| SSE 事件表、done、final/error 互斥 | 3, 4 |
| 409/404/400、LLM 失败保留 user、max_iterations | 3, 4 |
| 取消走 Request.Context | 3（`ctx.Err`）+ 4（传入 `c.Request.Context()`） |
| 4096 截断、key 不进响应 | 2 GetSession、3 send |
| 测试 fake LLM、无真网 | 1, 3, 5 |
| agent.yaml / llm.yaml | 6 |

**Placeholder scan:** Task 6 的 start 命令要求从 `app.go` 复制并列出全部替换项（pid/port/文案），不是空的 “implement later”。restart 若仓库已有 `framework/command/grpc` 的 restart，按其文件复制。

**Type consistency:** `contract.LLM`、`ChatResponse.Finish` 常量、`AgentEvent.Type/Data`、`MemoryAgent.Run` 签名在 Task 1–4 一致。`ErrLLMFailed` 只定义在 `contract/agent.go`。
