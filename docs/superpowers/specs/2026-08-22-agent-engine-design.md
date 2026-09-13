# Agent Engine 设计（ReAct + SSE）

日期：2026-08-22  
状态：待用户确认后进入实现计划

## 背景与目标

hade 现有 Kernel 提供两套引擎：HTTP（业务 Web）与 gRPC，分别由 `hade app start` 与 `hade grpc start` 拉起。本设计增加第三套 **Agent Engine**，在独立进程、独立端口上提供 ReAct 多轮运行时。客户端只订阅 SSE，不执行工具。

这里的 React 指 **ReAct（Reasoning + Acting）**，不是前端框架。

**第一版必须同时具备：**

- **内循环**：同一句用户问题内 Thought → Action → Observation，直到 Final Answer 或失败。
- **外循环**：`session` 保存对话历史，同一会话多次提问都带上历史再跑 ReAct。

**明确不做（第一版）：**

- 工具由客户端执行、再 POST 观察回来
- 模型 token 级流式（只对流向客户端的 Agent 事件做 SSE）
- SSE `Last-Event-ID` 断线重放
- 服务端主动 kill 某个 Run 的管理 API
- 把现有 ORM/Redis/SSH 等 Provider 注册成内置工具
- 真实 LLM 网络调用出现在自动化测试里

## 架构

启动路径与现有框架一致：`main.go` 创建容器、绑定 Provider、创建三套 Engine、注入 Kernel、再执行 Cobra。

```
main
  ├─ HadeContainer
  │    ├─ 已有 Provider（app/env/config/log/…）
  │    ├─ hade:llm     默认 OpenAI 兼容 HTTP
  │    └─ hade:agent   session + 工具表 + ReAct
  ├─ HttpEngine   → kernel.HttpEngine()  → hade app start
  ├─ GrpcEngine   → kernel.GrpcEngine()  → hade grpc start
  └─ AgentEngine  → kernel.AgentEngine() → hade agent start
```

AgentEngine 仍是 HTTP 服务（SSE 需要 HTTP），但是 **专用监听**：独立端口、独立 pid 文件、不托管 `dist` 静态资源、不挂业务 swagger。Kernel 与命令只负责进程和 Listen；ReAct 只存在于 `hade:agent`。

### 目录与职责

| 角色 | 路径 |
|------|------|
| 创建引擎并挂 Agent 路由 | `app/agent/kernel.go`（`NewAgentEngine`） |
| 示例工具注册 | `app/agent/tool/`（`echo`、`time`） |
| LLM 协议 | `framework/contract/llm.go`，Key `hade:llm` |
| Agent 协议 | `framework/contract/agent.go`，Key `hade:agent` |
| 默认实现 | `framework/provider/llm/`、`framework/provider/agent/` |
| 命令 | `framework/command/agent/`：`start` / `stop` / `restart` / `state` |
| Kernel | `HadeKernelService` 增加 `AgentEngine() http.Handler` |

pid 文件：`{runtime_folder}/agent.pid`，对齐 grpc 的 `{runtime_folder}/grpc.pid`。

默认监听端口：`8889`（与业务 HTTP 默认 `8888` 错开），由配置 `agent.port` 覆盖。

## 协议

### `hade:llm`

只做带工具的一轮补全。不感知 session，不执行工具。

```text
Chat(ctx, req ChatRequest) (ChatResponse, error)

ChatRequest:
  Messages []Message     // role: system | user | assistant | tool
  Tools    []ToolSpec    // Name, Description, Parameters（JSON Schema）
  可选：MaxTokens、Temperature

ChatResponse:
  Message   assistant 文本（可为空）
  ToolCalls []ToolCall   // ID、Name、Arguments（JSON 字符串）
  Finish    stop | tool_calls | length
```

默认 Provider：OpenAI 兼容 HTTP。配置：

- `llm.base_url`
- `llm.api_key`
- `llm.model`

业务可替换 Provider。第一版 `Chat` 为一次完整响应，不在 LLM 协议上暴露 token 流。

### 工具（挂在 `hade:agent`，不占用新的容器 Key）

```text
ToolSpec: Name, Description, Parameters（JSON Schema）
Handler:  func(ctx, argsJSON string) (observation string, err error)

RegisterTool(spec, handler)
ListTools() []ToolSpec
```

框架随 Agent 进程注册两个示例：

- `echo`：把参数中的文本原样返回
- `time`：返回当前 UTC 时间的 RFC3339 字符串

Handler 在 Agent 进程内同步执行。禁止示例工具执行 shell 或读取任意文件。

业务在 `NewAgentEngine` 或独立初始化函数里 `RegisterTool`。

### `hade:agent`

```text
CreateSession(ctx) (sessionID string, error)
GetSession(ctx, id) (Session, error)
Run(ctx, sessionID, userMessage string, events chan<- AgentEvent) error
```

`Session` 至少包含：`ID`、按时间排列的消息历史（user / assistant / tool 观察）。  
`sessionID` 使用 UUID。  
第一版存储为 **进程内存**；`Run` 不依赖具体存储，后续可改为 cache/redis 而不改循环。

同一 `sessionID` 上 `Run` **串行**：session 级互斥。第二个并发 `Run` 在进入循环前失败，由 Engine 映射为 HTTP 409，不升级为 SSE。

`Run` 内循环：

1. 将本轮 `userMessage` 追加到历史。
2. 调用 `hade:llm.Chat`（历史 + `ListTools()`）。
3. 发出 `thought`（assistant 文本；无文本则 `content` 为空字符串，事件仍发）。
4. 若存在 `ToolCalls`：对每个 call 依次发 `action`，执行 handler，发 `observation`，把 tool 结果写入历史，回到步骤 2。
5. 若无工具且 `Finish=stop`：发 `final`，结束。
6. 若达到 `agent.max_iterations`（默认 8）：发 `error`（`max_iterations`），结束。不发 `final`。

`events` 由 Engine 消费并写成 SSE。**由 Engine 在 `Run` 返回后关闭 channel**，`Run` 只负责往 channel 发送事件，避免双关。

## HTTP 与 SSE

客户端只访问 Agent 端口。

| 方法 | 路径 | 响应 |
|------|------|------|
| `POST` | `/sessions` | `201` JSON `{ "id": "<uuid>" }` |
| `GET` | `/sessions/:id` | `200` JSON 会话摘要与历史；不存在 `404` |
| `POST` | `/sessions/:id/messages` | 成功则 `text/event-stream`；失败且未升流则 JSON |

`POST /sessions/:id/messages` 请求体：

```json
{ "message": "现在几点？" }
```

`message` 为空或缺失：`400` JSON，不升流。

SSE：

- `Content-Type: text/event-stream`
- 每条：`event: <类型>`，`data: <一行 JSON>`
- 每条带单调递增 `id` 字段（事件序号，从 1 开始），第一版不实现断线重放

| event | data | 何时 |
|--------|------|------|
| `session` | `{ "session_id": "..." }` | 升流后第一条 |
| `thought` | `{ "content": "..." }` | 每次 `Chat` 返回后 |
| `action` | `{ "name": "...", "arguments": ... }` | 每个 tool call |
| `observation` | `{ "name": "...", "content": "..." }` | 每个 handler 返回后 |
| `final` | `{ "content": "..." }` | 内循环成功结束 |
| `error` | `{ "code": "...", "message": "..." }` | 失败 |
| `done` | `{}` | 成功或失败都发，然后关闭连接 |

同一轮多个 tool call：按顺序多次 `action` → `observation`，再进入下一轮 `thought`。

`final` 与 `error` 互斥。两者之后都跟 `done`。历史不通过 SSE 全量回放；需要上下文用 `GET /sessions/:id`。

`action.arguments` 使用 JSON 对象（已解析的参数）。若模型给出非法 JSON，则 `arguments` 为 `{}`，并把原始字符串放进随后的 `observation.content` 作为错误文本，不中断循环。

## 错误、取消、安全

| 场景 | 行为 |
|------|------|
| session 不存在 | 不升流，`404` JSON |
| 同 session 并发 Run | 不升流，`409` JSON |
| body 非法 / message 空 | 不升流，`400` JSON |
| LLM 失败（网络、非 2xx、响应无法解析） | 已升流：`error` code `llm_failed` + `done`。本轮 **保留已写入的 user 消息**，不写入失败的 assistant 消息 |
| tool handler 返回 error | 不中断循环；`observation.content` 为错误文本，截断到 4096 字节 |
| tool 或 LLM 使用 `ctx` 超时 | 同失败路径：tool 超时 → observation 文本；LLM 超时 → `llm_failed` |
| `max_iterations` | `error` code `max_iterations` + `done`；已产生的 thought/action/observation 留在历史 |
| handler/Engine panic | gin recovery；已升流则 `error` code `internal` + `done`，并写日志 |

**取消：** 客户端断开 SSE → `Request.Context()` cancel → `Run` / `Chat` / tool 停止。能写响应则 `error` code `canceled` + `done`；来不及写则只记日志。取消后已写入历史的 user 与已完成的 tool 观察保留。

**安全：** `llm.api_key` 不得出现在 SSE 或 `GET /sessions` 响应中。thought / observation / final 的 `content` 在写入 SSE 前截断到 4096 字节；完整内容可留在内存历史中，但 `GET /sessions` 同样截断到 4096，避免调试接口变成大对象出口。

## 配置

新增配置文件 `config/{env}/agent.yaml`（或并入 `app.yaml` 的 `agent` 段）。第一版使用 **`config/{env}/agent.yaml`**，避免把 Agent 监听和业务 HTTP 混在一个文件里。

```yaml
port: 8889
max_iterations: 8
```

LLM 使用 `config/{env}/llm.yaml`：

```yaml
base_url: "https://api.openai.com/v1"
api_key: ""
model: "gpt-4o-mini"
```

空 `api_key` 时：进程可以启动，但第一次 `Chat` 失败并走 `llm_failed`（便于先测 SSE 与 fake LLM）。

## 测试

原则：测协议与循环，不测真实模型。

- 提供可注入的 fake `hade:llm`：按预设脚本依次返回「带 tool_call 的响应」和「stop + 文本」。
- `Run` 单测：`echo` 走通 thought → action → observation → final；断言历史中有 user、assistant、tool 观察；`max_iterations` 时只有 `error` 没有 `final`。
- 同一 session 并发第二次 `Run` 返回错误（供 Engine 映射 409）。
- HTTP：`httptest` 访问 `POST /sessions/:id/messages`，按事件名断言顺序：`session` → `thought` → `action` → `observation` → `thought` → `final` → `done`。
- OpenAI 兼容 Provider：只测请求/响应 JSON 编解码（fixture），不发起真实 HTTP。
- 不测：`hade agent start` 守护进程、真实 LLM、断线重放。

## 实现顺序（供后续计划拆分）

1. 协议与空实现（contract + fake llm + 内存 session）
2. ReAct `Run` 与示例工具
3. AgentEngine 路由与 SSE
4. Kernel / `main` / `hade agent` 命令与配置
5. 默认 OpenAI 兼容 Provider（可配置，测试用 fixture）
