# Agent SessionStore：内存 + Redis

日期：2026-08-30  
状态：待用户确认后进入实现计划

## 背景与目标

`MemoryAgent` 目前用进程内 `map[string]*memorySession` 存历史，并用两把 Go mutex（`runMu` / `dataMu`）保证「同一 Session 同时只能一轮 Run」以及读写互斥。进程退出即丢失；多实例无法共享。

本设计抽出 `SessionStore`：默认内存实现保持现有单测与无 Redis 场景；绑定 `hade:redis` 时用 Redis 持久化，并支持多 Agent 进程共享。连接通过 **`contract.RedisService.GetClient()` 拿一次 `*redis.Client`**，不把 `RedisService` 散落到 Run 热路径。

**成功标准：**

- `contract.Agent` 对外行为不变（错误 sentinel、SSE、GetSession 在 Run 中途可读）。
- 未绑定 Redis 时行为与今天一致（内存 store）。
- 绑定 Redis 后：重启进程仍能 `GetSession`；两个进程对同一 id 同时 `Run` 时一方得到 `ErrSessionBusy`。
- Redis store 单测用 `github.com/alicebob/miniredis/v2`，不连真实 Redis。

## 明确不做

- 不改 `contract.Agent` / ReAct / HTTP 协议。
- 不把工具表存 Redis（`RegisterTool` 仍是进程内；多实例必须各自走同一套 kernel 注册）。
- 不走 `contract.Cache`（缺锁、Lua、集合计数）。
- 不做 Session 过期 TTL / 后台清理（仍靠 `MaxSessions`）。
- 不把 `MemoryAgent` 改名。
- 不提交真实 Redis 地址或密码。

## 包与职责

仍在 `framework/provider/agent`，不新建 framework 包。

| 路径 | 职责 |
|------|------|
| `store.go` | `SessionStore` / `RunSession` 接口 |
| `session.go` | 现有 `memorySession` 改为内存 `RunSession`；`memoryStore` |
| `redis_store.go` | Redis 实现：文档、集合、锁、Lua |
| `service.go` | `MemoryAgent` 持有 `SessionStore`，不再持有 `sess` map |
| `provider.go` | 有 `LogKey` 则注入 Log；有 `RedisKey` 则 `GetClient()` 注入 Redis store，否则内存 store |
| `session_test.go` / `redis_store_test.go` | 内存行为保持；Redis 用 miniredis |

## 接口

```go
type SessionStore interface {
    Create(ctx context.Context) (id string, err error)
    // Open 只读。Session 不存在返回 contract.ErrSessionNotFound。
    Open(ctx context.Context, id string) (messages []contract.Message, err error)
    // TryBeginRun 抢 Run 锁。Busy / NotFound 与今天语义一致。
    // 调用方必须 Release（defer）。
    TryBeginRun(ctx context.Context, id string) (RunSession, error)
}

type RunSession interface {
    ID() string
    Snapshot() []contract.Message
    Length() int
    UsedBytes() int
    AppendWithin(limit, reserve int, msgs ...contract.Message) error
    AppendReserved(msgs ...contract.Message)
    TruncateTo(n int)
    Release()
}
```

字节配额、`settleReserveBytes`、`cloneMessages` 仍在本包，Redis 与内存共用同一套计算，避免两套上限语义。

`GetSession` 的对外截断（`ContentMaxBytes`）仍在 `MemoryAgent.GetSession`，不进 store。

## Redis 数据模型

| Key | 类型 | 内容 |
|-----|------|------|
| `hade:agent:session:{id}` | string | JSON：`{"messages":[...],"bytes":n}` |
| `hade:agent:sessions` | set | 全部 session id（`SCARD` 做 `MaxSessions`） |
| `hade:agent:lock:{id}` | string | Run 锁，值为随机 token |

**Create：** `SCARD` 已达上限 → `ErrSessionLimit`。否则 UUID，`SET NX` 空文档，`SADD`。任一步失败回滚（`DEL` + `SREM`）。

**Open：** `GET` 文档；`redis.Nil` → `ErrSessionNotFound`。

**TryBeginRun：** Session 必须已存在。`SET lock NX PX 60000`，失败 → `ErrSessionBusy`。成功后启动续期（约 20s 一次，`PEXPIRE` 且仅当值仍为本人 token）。`Release` 停续期，Lua：值等于 token 才 `DEL`。

**写文档：** Lua：锁 token 匹配才 `SET` 文档。不匹配视为僵尸写，返回 `ErrInternal`（不把细节给客户端）。内存路径无此检查。

**崩溃残留：** 锁最多 60s 过期。下一轮 `TryBeginRun` 成功后、写入 user 消息前，若历史末尾是未闭环的 `assistant.tool_calls`，走现有 `settlePending(settleReasonInternal)`，再继续本轮。

Redis 命令失败（连接断开等）→ `ErrInternal`。

## Provider 装配

`Params` 在现有 `llm, maxIter, limits, logger` 上增加第五项 `SessionStore`：

- `c.IsBind(contract.RedisKey)`：`GetClient()`（默认 `redis` 配置路径，与 `GetBaseConfig` 一致）。`GetClient` 失败则 `NewHadeAgentService` 返回 error，容器 Make 失败。生产已绑 Redis 却连不上时，禁止静默退回内存（多实例会各写各的）。
- 未绑定 Redis：内存 store（单测、库直用）。

`NewHadeAgentService`：第 5 个参数类型断言为 `SessionStore`；缺省则 `newMemoryStore(limits)`。

`limits.MaxSessions` 传给 store 构造函数。

## 测试

### 内存

现有 `service_test.go` 继续用 `NewMemoryAgent`（内部内存 store）。断言 `a.sess[id]` 的用例改为经 `GetSession` / store 或测试辅助方法，不穿透 Redis。

### Redis（miniredis）

依赖：`github.com/alicebob/miniredis/v2`（测试依赖，随 `go test` 拉取）。

每个用例 `miniredis.Run()`，`redis.NewClient(&redis.Options{Addr: s.Addr()})`，构造 `newRedisStore(client, limits)`。

必测：

1. Create + Open 往返；进程内再 new 一个 store 共用同一 miniredis，仍能 Open（模拟重启/第二实例）。
2. `MaxSessions` → `ErrSessionLimit`。
3. 同一 id 第二次 `TryBeginRun` → `ErrSessionBusy`；`Release` 后可再抢。
4. 两个 `redisStore` 实例（两个 client 连同一 miniredis）交叉 Busy。
5. `AppendWithin` 超限 → `ErrHistoryLimit`；`TruncateTo` 回滚条数与 bytes。
6. 伪造错误 token 写文档被 Lua 拒绝。
7. `miniredis` 关闭后命令失败 → `ErrInternal`（或 Create/Open 失败）。
8. 锁 key 手动 `DEL` 模拟过期后，第二 store 能 `TryBeginRun`。

**不测：** 真实 20s 续期墙钟（用可注入的短 TTL 或对续期函数单测即可，避免 `time.Sleep(20s)`）。

Provider：可用假 `RedisService`，`GetClient` 返回连 miniredis 的 client，断言 `NewHadeAgentService(Params...)` 用的是 Redis store。

## 错误与 HTTP

sentinel 不变，`agenthttp` 映射不变。新增失败都进 `ErrInternal`，不新增 contract 错误类型。

## 风险

- 锁 TTL 与超长 LLM：靠续期；续期 goroutine 必须随 `Release` / `ctx.Done` 退出，避免泄漏。
- 多实例工具不一致：文档与启动约定，本设计不解决。
- miniredis 与 go-redis v8 的 Lua/SET NX 语义需在实现时用最小脚本验证；若某命令不支持，改为等价的 GET+SET 事务并在测试中锁死行为。
