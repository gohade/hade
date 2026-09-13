# Agent Session 滑动窗口（token 超限改写历史）

日期：2026-08-30  
状态：待用户确认后进入实现计划

## 背景与目标

`AgentRuntime.loop` 每次 `llm.Chat` 都把 `RunSession.Snapshot()` 整段送出。Session 只按字节记账（`UsedBytes` / `messageBytes`），没有 token 估算，也没有超限压缩。多轮 ReAct 与长工具观察会把上下文撑到模型上限。

本设计在 **每次 Chat 前** 用真实 tokenizer 计算即将送出的 messages 的 token 数；若超过配置阈值，从头部丢掉完整旧对话轮次并 **写回 Session**（内存与 Redis），直到低于阈值或只剩当前轮。当前轮仍超限则不删、本轮 Chat 失败。

**成功标准：**

- `max_context_tokens > 0` 时，Chat 前历史被滑动窗口改写；`GetSession` 与后续 Run 看到的是压缩后的消息。
- 不拆开 `assistant.tool_calls` 与对应 `tool` 回复；`system` 始终保留。
- 当前轮（本轮 Run 追加的 user 及其后消息）仍超限 → 可丢的旧轮照常 `Replace` 掉，当前轮保留；`ErrContextTooLong`，SSE `code: context_too_long`，不调用 LLM。
- `max_context_tokens <= 0` 或未配置：行为与现在完全一致。
- 单测用离线 tokenizer，不打真实 LLM / 不访问下载词表的网络。

## 明确不做

- 不做 LLM 摘要压缩，不截短单条 content（除非走现有 GetSession/SSE 的 `ContentMaxBytes`，与本功能无关）。
- 不把 `ChatRequest.Tools` JSON Schema 计入阈值。
- 不给 `contract.LLM` 增加 `Model()`；不引入 DeepSeek 官方词表文件。
- 不改 ReAct 事件类型集合；不改「同一 Session 同时一轮 Run」的锁语义。
- 自动化测试不调用真实 DeepSeek / OpenAI。

## 架构

```
Run.loop
  Append(user)                // 已有
  EventSession                // 已有，压缩发生在此之后、Chat 之前
  for each iteration:
    compactIfNeeded(session)  // 新增：token 计数 + 丢旧轮 + Replace
    llm.Chat(Snapshot, Tools)
    ... tool / final ...
```

压缩发生在已推 `session` 事件之后，与现有 `llm_failed` 相同：失败走 SSE `error`，流通常已是 HTTP 200，**不**走 `writeAgentError` 的 JSON 400。

### 组件

| 单元 | 路径 | 职责 |
|------|------|------|
| 配置 | `config/{env}/agent.yaml` 的 `max_context_tokens` | 阈值；`<=0` 关闭 |
| 模型名 | Provider 读 `llm.model`，注入 `AgentRuntime` | 只用于选 encoding |
| encoding 映射 | `framework/provider/agent` 内纯函数 | `model` → tiktoken encoding 名 |
| tokenizer | 可 embed / 离线加载的 tiktoken Go 库 | 对字符串 `Encode` 计 token |
| 窗口算法 | 同包纯函数 `compactMessages` | 切轮次、丢最旧、保护当前轮 |
| 写回 | `RunSession.Replace` | 内存改切片并重算 bytes；Redis 用现有锁 token 整份 JSON |
| 错误 | `contract.ErrContextTooLong` + SSE / `errorEventData` | 当前轮仍超限 |

`NewAgentRuntime(llm, maxIter)` 保持签名：默认 `maxContextTokens=0`（关闭），现有单测不用改。`NewHadeAgentService` / Provider 在构造后写入 `model` 与 `maxContextTokens`。

## 轮次与当前轮

- **system**：所有 `role==system` 的消息固定留在列表最前（相对顺序不变），不参与丢弃。
- **一轮**：一条 `user`，加上它后面直到下一条 `user`（不含）之间的全部 `assistant` / `tool`（含多轮 tool_calls）。
- **丢弃**：整轮一起从头部去掉（system 之后最旧的一轮）。禁止只删一半 tool 闭环。
- **当前轮**：本轮 `Run` 在 `loop` 开头 `Append` 的那条 user 的下标，以及该下标起到末尾的所有消息。压缩不得删除下标 `>= currentUserIndex` 的消息。
- 列表开头若出现无前置 user 的 `assistant`/`tool`（异常历史）：视为不可拆前缀，与 system 一样保留，直到只能动「完整旧 user 轮次」。

`compactMessages(messages, currentUserIndex, maxTokens, countFn)`：

1. 若 `countFn(messages) <= maxTokens`，原样返回。
2. 否则反复去掉 system/前缀之后、`currentUserIndex` 之前的最旧完整 user 轮；每丢掉一轮后 `currentUserIndex` 前移。
3. 无法再丢且仍超限：返回 **已丢掉所有可丢旧轮之后** 的消息（通常只剩 system + 当前轮）以及 `ErrContextTooLong`。调用方 **先 `Replace` 再** 返回该错误，避免失败时仍把整段旧历史留在 Session。若 `Replace` 失败则走 `persistFailed`，不再额外删当前轮。

## Tokenizer 与计数

库：优先选用 **离线 embed 词表** 的实现（例如 `github.com/tiktoken-go/tokenizer`，或 `pkoukk/tiktoken-go` 的 offline loader）。CI 与单测禁止联网拉 BPE。

`encodingForModel(model string)`（大小写不敏感，按前缀）：

| 条件 | encoding |
|------|----------|
| `gpt-4o` / `gpt-4.1` / `o1` / `o3` 前缀 | `o200k_base` |
| `gpt-4` / `gpt-3.5` 前缀 | `cl100k_base` |
| `deepseek-` 前缀 | `cl100k_base`（真实 BPE，**不是** DeepSeek 官方词表） |
| 空或其它 | `cl100k_base` |

映射只改这一处函数。以后若嵌入 DeepSeek 词表，只换 `deepseek-` 行。

单条 `messageTokens`：对 `Role`、`Content`、`ToolCallID`、每个 `ToolCall` 的 `ID`/`Name`/`Arguments` 分别 `Encode` 后相加，再加常量 **每条消息 +3**（对齐 OpenAI chat 估算，不追求与上游 `usage.prompt_tokens` 逐 token 一致）。列表 token = 各消息之和。

Encoding 加载失败：本轮按 **内部错误** 处理（`ErrInternal` + 现有 internal SSE），不静默跳过压缩（避免以为已裁窗实际原样撑爆）。

进程内按 encoding 名缓存 tokenizer 实例。

## 配置

`config/development/agent.yaml`、`production`、`testing` 同步增加：

```yaml
port: ":8889"
max_iterations: 8
max_context_tokens: 32000
```

- 缺省键或 `<= 0`：关闭压缩。
- Provider：`agent.max_context_tokens` 与现有 `agent.max_iterations` 一样从 Config 读取；`llm.model` 已有，注入运行时。
- 推荐默认 **32000**，给补全和未计入的 Tools schema 留余量。

## `RunSession.Replace`

```go
// Replace 用 messages 整表替换历史并重算 UsedBytes。
// Redis 写失败返回 contract.ErrInternal。不得在未持 Run 锁时调用。
Replace(messages []contract.Message) error
```

不复用 `TruncateTo`（那是从 **尾部** 回滚未闭环 tool_calls）。`Replace` 后 `Length`/`Snapshot`/`UsedBytes` 与新切片一致。`cloneMessages` 写入，避免调用方与 store 共享底层数组。

Redis：与 `Append` 相同，Lua 校验锁 token 后 `SET` 整份 `sessionDoc`。

## 错误与 SSE

- `contract.ErrContextTooLong`（sentinel 文本 `context_too_long`）。
- `loop` 在 compact 返回该错误时：若结果切片与压缩前不同则先 `Replace`；不调用 LLM；`send`/`tryEmit` `EventError`，`code`/`message` 为 `context_too_long`；函数返回该 sentinel。
- `framework/agenthttp/handler.go` 的 `errorEventData` 增加 `ErrContextTooLong` 分支。无需改 `writeAgentError`（压缩在 `EventSession` 之后，走 SSE）。
- `Replace` 失败：走现有 `persistFailed` / `ErrInternal`。

## 数据流（一次提问）

1. `Append` 新 user（计入当前轮起点）。
2. SSE `session`。
3. 计 token；超限则丢旧轮并 `Replace`；仍超限则 SSE error 并返回。
4. `Chat(Snapshot(), Tools)`。
5. 工具路径 `Append` assistant + tool 后，下一轮迭代回到步骤 3（当前轮已包含新观察）。

## 测试

- `encodingForModel` 表驱动。
- `compactMessages`：不超限不改；丢最旧整轮后低于阈值；保留 system；不拆 tool_calls；`currentUserIndex` 之后不动；无法压缩时 error 且切片不变。
- `messageTokens`：短 ASCII 字符串用真实 tokenizer，断言为正且随内容变长。
- `Run`（`maxContextTokens` 很小、ScriptLLM）：先人工写入两轮旧对话再 `Run` 一句新话；Chat 入参与 `GetSession` 均不含被丢掉的最旧轮；ScriptLLM 不触网。
- 当前轮单独超限：`Run` 返回 `ErrContextTooLong`；可丢的旧轮已 `Replace` 掉；新 user（当前轮）仍在 Session。
- `maxContextTokens==0`：不调用 compact / 不 `Replace`。
- 内存与 Redis（miniredis）的 `Replace`：替换后 Open/Snapshot 一致，`UsedBytes` 等于 `messageBytes` 之和。
- HTTP：`errorEventData` 对 `ErrContextTooLong` 的 SSE 单测（对齐现有 llm_failed 表驱动）。

## 实现顺序建议

1. `ErrContextTooLong`、`Replace`、encoding/token 纯函数与单测。
2. `compactMessages` + `loop` 接入 + Provider 配置。
3. SSE `errorEventData`；development/production/testing 的 `agent.yaml`。
