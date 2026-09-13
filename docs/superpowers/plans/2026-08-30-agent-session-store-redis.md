# Agent SessionStore Redis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 抽出 `SessionStore`，默认内存；绑定 `hade:redis` 时用 Redis 持久化 Session，并支持多实例互斥 Run。

**Architecture:** `MemoryAgent` 只持有 `SessionStore`。内存实现搬现有 `memorySession` 锁语义。Redis 实现用一份 JSON 文档 + set 计数 + token 锁（Lua 校验）。`contract.RedisService.GetClient()` 只在 Provider 构造时调用一次。

**Tech Stack:** Go 1.25、`github.com/go-redis/redis/v8`、`github.com/alicebob/miniredis/v2`、goconvey。

## Global Constraints

- 不改 `contract.Agent`、ReAct、SSE、HTTP 映射。
- 不把工具表存 Redis。
- 不走 `contract.Cache`。
- 不做 Session TTL / 后台清理。
- 不把 `MemoryAgent` 改名。
- Redis 单测只用 miniredis，不连真实 Redis。
- 绑了 Redis 但 `GetClient` 失败：`NewHadeAgentService` 返回 error，禁止静默退回内存。
- 按用户要求：**实现过程中不要 git commit**，除非用户明确说提交。
- 不要改无关文件。

## File structure

| Path | Responsibility |
|------|----------------|
| `framework/provider/agent/store.go` | `SessionStore` / `RunSession` 接口 |
| `framework/provider/agent/session.go` | `memoryStore` + `memorySession`（实现 `RunSession`，增加 `Release`） |
| `framework/provider/agent/redis_store.go` | Redis store / 锁 / Lua |
| `framework/provider/agent/service.go` | `MemoryAgent.store`；Create/Get/Run 走 store |
| `framework/provider/agent/provider.go` | Params 第 5 项：store 或 `GetClient` error |
| `framework/provider/agent/session_test.go` | 内存 store 接口测试 |
| `framework/provider/agent/redis_store_test.go` | miniredis 测试 |
| `framework/provider/agent/service_test.go` | 去掉 `a.sess[id]` 穿透；Busy 改走 `TryBeginRun` |
| `framework/provider/agent/provider_test.go` | Params 长度 5；假 RedisService + miniredis |

Spec：`docs/superpowers/specs/2026-08-30-agent-session-store-redis-design.md`

---

### Task 1: 接口 + 内存 store + MemoryAgent 改走 store

**Files:**
- Create: `framework/provider/agent/store.go`
- Create: `framework/provider/agent/session_test.go`
- Modify: `framework/provider/agent/session.go`
- Modify: `framework/provider/agent/service.go`
- Modify: `framework/provider/agent/service_test.go`

**Interfaces:**
- Consumes: 现有 `memorySession` 的 `snapshot` / `appendWithin` / `appendReserved` / `truncateTo` / `length` / `usedBytes`；`Limits.MaxSessions`。
- Produces:
  - `type SessionStore interface { Create(ctx context.Context) (string, error); Open(ctx context.Context, id string) ([]contract.Message, error); TryBeginRun(ctx context.Context, id string) (RunSession, error) }`
  - `type RunSession interface { ID() string; Snapshot() []contract.Message; Length() int; UsedBytes() int; AppendWithin(limit, reserve int, msgs ...contract.Message) error; AppendReserved(msgs ...contract.Message); TruncateTo(n int); Release() }`
  - `func newMemoryStore(maxSessions int) SessionStore`

- [ ] **Step 1: 写内存 store 失败测试**

`framework/provider/agent/session_test.go`：

```go
package agent

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryStore_CreateOpenLimitBusy(t *testing.T) {
	Convey("内存 store：创建、只读、上限、Busy", t, func() {
		store := newMemoryStore(1)
		ctx := context.Background()

		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		msgs, err := store.Open(ctx, id)
		So(err, ShouldBeNil)
		So(msgs, ShouldBeEmpty)

		_, err = store.Create(ctx)
		So(err, ShouldEqual, contract.ErrSessionLimit)

		_, err = store.Open(ctx, "missing")
		So(err, ShouldEqual, contract.ErrSessionNotFound)

		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		_, err = store.TryBeginRun(ctx, id)
		So(err, ShouldEqual, contract.ErrSessionBusy)
		run.Release()
		run2, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		run2.Release()
	})
}

func TestMemoryStore_AppendQuotaAndTruncate(t *testing.T) {
	Convey("内存 store：配额与 truncate", t, func() {
		store := newMemoryStore(8)
		ctx := context.Background()
		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		defer run.Release()

		So(run.AppendWithin(10, 0, contract.Message{Role: "user", Content: "abcdefghijk"}), ShouldEqual, contract.ErrHistoryLimit)
		So(run.AppendWithin(100, 0, contract.Message{Role: "user", Content: "hi"}), ShouldBeNil)
		So(run.Length(), ShouldEqual, 1)
		mark := run.Length()
		So(run.AppendWithin(100, 0, contract.Message{Role: "assistant", Content: "there"}), ShouldBeNil)
		run.TruncateTo(mark)
		So(run.Length(), ShouldEqual, 1)
		So(run.Snapshot()[0].Content, ShouldEqual, "hi")
	})
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./framework/provider/agent/ -count=1 -run 'TestMemoryStore_'`

Expected: FAIL，`newMemoryStore` undefined。

- [ ] **Step 3: 加接口并把 memorySession 收进 memoryStore**

`framework/provider/agent/store.go`：

```go
package agent

import (
	"context"

	"github.com/gohade/hade/framework/contract"
)

type SessionStore interface {
	Create(ctx context.Context) (id string, err error)
	Open(ctx context.Context, id string) (messages []contract.Message, err error)
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

在 `session.go` 增加（保留现有 `memorySession` 字段与 `append*` 实现，只加 `ID`/`Release` 和 `memoryStore`）：

```go
func (s *memorySession) ID() string { return s.id }

func (s *memorySession) Release() { s.runMu.Unlock() }

type memoryStore struct {
	mu          sync.RWMutex
	sess        map[string]*memorySession
	maxSessions int
}

func newMemoryStore(maxSessions int) SessionStore {
	if maxSessions <= 0 {
		maxSessions = DefaultLimits().MaxSessions
	}
	return &memoryStore{sess: map[string]*memorySession{}, maxSessions: maxSessions}
}

func (m *memoryStore) Create(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sess) >= m.maxSessions {
		return "", contract.ErrSessionLimit
	}
	id := uuid.New().String()
	m.sess[id] = newMemorySession(id)
	return id, nil
}

func (m *memoryStore) Open(ctx context.Context, id string) ([]contract.Message, error) {
	m.mu.RLock()
	session, ok := m.sess[id]
	m.mu.RUnlock()
	if !ok {
		return nil, contract.ErrSessionNotFound
	}
	messages := session.snapshot()
	if messages == nil {
		messages = []contract.Message{}
	}
	return messages, nil
}

func (m *memoryStore) TryBeginRun(ctx context.Context, id string) (RunSession, error) {
	m.mu.RLock()
	session, ok := m.sess[id]
	m.mu.RUnlock()
	if !ok {
		return nil, contract.ErrSessionNotFound
	}
	if !session.runMu.TryLock() {
		return nil, contract.ErrSessionBusy
	}
	return session, nil
}
```

`session.go` 需 import `context` 与 `github.com/google/uuid`（若尚未有）。

- [ ] **Step 4: 再跑 TestMemoryStore，应通过**

Run: `go test ./framework/provider/agent/ -count=1 -run 'TestMemoryStore_'`

Expected: PASS。

- [ ] **Step 5: MemoryAgent 改用 store**

`MemoryAgent` 删除 `sessMu` / `sess`，增加：

```go
store SessionStore
```

`NewMemoryAgentWithLimits`：

```go
return &MemoryAgent{
	llm:     llm,
	maxIter: maxIter,
	limits:  limits.normalize(),
	store:   newMemoryStore(limits.normalize().MaxSessions),
	sess:    nil, // 删除 sess 字段
}
```

`NewHadeAgentService` 在现有 logger 之后：

```go
if len(params) > 4 {
	if store, ok := params[4].(SessionStore); ok && store != nil {
		agent.store = store
	}
}
```

`CreateSession` 变为 `return a.store.Create(ctx)`。

`GetSession`：

```go
messages, err := a.store.Open(ctx, id)
if err != nil {
	return contract.Session{}, err
}
if messages == nil {
	messages = []contract.Message{}
}
for i := range messages {
	messages[i] = truncateMessageForRead(messages[i], contract.ContentMaxBytes)
}
return contract.Session{ID: id, Messages: messages}, nil
```

`Run` 里查找 + `TryLock` 换成：

```go
session, err := a.store.TryBeginRun(ctx, sessionID)
if err != nil {
	return err
}
defer session.Release()

run := &runState{agent: a, session: session, events: events, ctx: ctx}
run.settleDanglingToolCalls()
// recover defer 不变
return run.loop(sessionID, userMessage)
```

`runState.session` 类型改为 `RunSession`。所有 `r.session.appendWithin` 等方法名已对齐接口。

`settleDanglingToolCalls`（写在 `service.go`，Run 写入 user 消息之前调用）：

```go
func (r *runState) settleDanglingToolCalls() {
	messages := r.session.Snapshot()
	pending := danglingToolCallIDs(messages)
	if len(pending) == 0 {
		return
	}
	r.pending = pending
	r.settlePending(settleReasonInternal)
}

func danglingToolCallIDs(messages []contract.Message) []string {
	answered := map[string]struct{}{}
	for _, message := range messages {
		if message.Role == "tool" && message.ToolCallID != "" {
			answered[message.ToolCallID] = struct{}{}
		}
	}
	var pending []string
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		for _, call := range message.ToolCalls {
			if _, ok := answered[call.ID]; !ok {
				pending = append(pending, call.ID)
			}
		}
	}
	return pending
}
```

方法大小写：`RunSession` 接口用 `Snapshot`/`Length`/`UsedBytes`/`AppendWithin`/`AppendReserved`/`TruncateTo`。`memorySession` 现有小写方法可改成导出方法（推荐直接把 `snapshot` 改名为 `Snapshot` 等，本包其它调用一并改），避免一对包装。

- [ ] **Step 6: 改 service_test 去掉 `a.sess`**

`TestRun_SessionBusy`：

```go
run, err := a.store.TryBeginRun(context.Background(), id)
So(err, ShouldBeNil)
ch := make(chan contract.AgentEvent, 2)
err = a.Run(context.Background(), id, "two", ch)
run.Release()
So(err, ShouldEqual, contract.ErrSessionBusy)
```

`usedBytes` 断言改为：

```go
msgs, err := a.store.Open(context.Background(), id)
So(err, ShouldBeNil)
used := 0
for _, message := range msgs {
	used += messageBytes(message)
}
So(used, ShouldBeLessThanOrEqualTo, 70)
```

内部未截断 arguments：

```go
msgs, err := a.store.Open(context.Background(), id)
So(err, ShouldBeNil)
So(len(msgs[1].ToolCalls[0].Arguments), ShouldEqual, len(hugeArguments))
```

增加一条 dangling settle 测试（可放 `service_test.go`）：

```go
func TestRun_SettlesDanglingToolCallsBeforeUserMessage(t *testing.T) {
	Convey("上一轮崩溃残留的 tool_calls 在本轮 Run 开始时补齐", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		held, err := a.store.TryBeginRun(context.Background(), id)
		So(err, ShouldBeNil)
		So(held.AppendWithin(a.limits.MaxHistoryBytes, settleReserveBytes([]contract.ToolCall{{ID: "c1"}}), contract.Message{
			Role:      "assistant",
			Content:   "thinking",
			ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}},
		}), ShouldBeNil)
		held.Release()

		events := make(chan contract.AgentEvent, 8)
		err = a.Run(context.Background(), id, "next", events)
		close(events)
		So(err, ShouldBeNil)
		session := mustSession(a, id)
		var toolMsgs int
		for _, message := range session.Messages {
			if message.Role == "tool" && message.ToolCallID == "c1" {
				toolMsgs++
				So(message.Content, ShouldEqual, settleReasonInternal)
			}
		}
		So(toolMsgs, ShouldEqual, 1)
	})
}
```

`fakeLLM` 需能对 "next" 直接 stop（现有 `fakeLLM` 若已返回 stop 即可）。

- [ ] **Step 7: 跑 agent provider 测试**

Run: `go test ./framework/provider/agent/ -count=1`

Expected: PASS。

- [ ] **Step 8: Commit**

跳过（用户未要求提交）。

---

### Task 2: Redis store + miniredis

**Files:**
- Create: `framework/provider/agent/redis_store.go`
- Create: `framework/provider/agent/redis_store_test.go`
- Modify: `go.mod` / `go.sum`（增加 `github.com/alicebob/miniredis/v2`）

**Interfaces:**
- Consumes: Task 1 的 `SessionStore` / `RunSession`；`*redis.Client`；`Limits.MaxSessions`。
- Produces:
  - `func newRedisStore(client *redis.Client, maxSessions int) SessionStore`
  - `func newRedisStoreWithLock(client *redis.Client, maxSessions int, lockTTL, renewEvery time.Duration) SessionStore`（测试注入短 TTL；`renewEvery==0` 表示不启续期 goroutine）

常量（`redis_store.go`）：

```go
const (
	sessionsSetKey = "hade:agent:sessions"
	lockTTLDefault = 60 * time.Second
	renewDefault   = 20 * time.Second
)

func sessionKey(id string) string { return "hade:agent:session:" + id }
func lockKey(id string) string    { return "hade:agent:lock:" + id }

type sessionDoc struct {
	Messages []contract.Message `json:"messages"`
	Bytes    int                `json:"bytes"`
}
```

Lua：

```go
const luaUnlock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

const luaPersist = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call("SET", KEYS[2], ARGV[2])
return 1
`

const luaRenew = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`
```

- [ ] **Step 1: 加 miniredis 依赖**

Run: `go get github.com/alicebob/miniredis/v2`

Expected: `go.mod` 出现该模块。

- [ ] **Step 2: 写 redis_store_test.go（先让编译失败）**

```go
package agent

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func miniClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return s, client
}

func TestRedisStore_CreateOpenAcrossStores(t *testing.T) {
	Convey("同一 miniredis 上两个 store 能读到同一 session（模拟重启/第二实例）", t, func() {
		_, client := miniClient(t)
		ctx := context.Background()
		a := newRedisStore(client, 8)
		id, err := a.Create(ctx)
		So(err, ShouldBeNil)

		b := newRedisStore(client, 8)
		msgs, err := b.Open(ctx, id)
		So(err, ShouldBeNil)
		So(msgs, ShouldBeEmpty)
	})
}

func TestRedisStore_MaxSessions(t *testing.T) {
	Convey("SCARD 达上限返回 ErrSessionLimit", t, func() {
		_, client := miniClient(t)
		store := newRedisStore(client, 1)
		ctx := context.Background()
		_, err := store.Create(ctx)
		So(err, ShouldBeNil)
		_, err = store.Create(ctx)
		So(err, ShouldEqual, contract.ErrSessionLimit)
	})
}

func TestRedisStore_BusyAndRelease(t *testing.T) {
	Convey("同一 id 二次 TryBeginRun 为 Busy，Release 后可再抢", t, func() {
		_, client := miniClient(t)
		store := newRedisStoreWithLock(client, 8, time.Minute, 0)
		ctx := context.Background()
		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		_, err = store.TryBeginRun(ctx, id)
		So(err, ShouldEqual, contract.ErrSessionBusy)
		run.Release()
		run2, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		run2.Release()
	})
}

func TestRedisStore_BusyAcrossStores(t *testing.T) {
	Convey("两个 store 实例交叉 Busy", t, func() {
		_, client := miniClient(t)
		ctx := context.Background()
		a := newRedisStoreWithLock(client, 8, time.Minute, 0)
		b := newRedisStoreWithLock(client, 8, time.Minute, 0)
		id, err := a.Create(ctx)
		So(err, ShouldBeNil)
		run, err := a.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		_, err = b.TryBeginRun(ctx, id)
		So(err, ShouldEqual, contract.ErrSessionBusy)
		run.Release()
	})
}

func TestRedisStore_AppendQuotaAndTruncate(t *testing.T) {
	Convey("AppendWithin 超限与 TruncateTo", t, func() {
		_, client := miniClient(t)
		store := newRedisStoreWithLock(client, 8, time.Minute, 0)
		ctx := context.Background()
		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		defer run.Release()
		So(run.AppendWithin(5, 0, contract.Message{Role: "user", Content: "too-long"}), ShouldEqual, contract.ErrHistoryLimit)
		So(run.AppendWithin(100, 0, contract.Message{Role: "user", Content: "ok"}), ShouldBeNil)
		mark := run.Length()
		So(run.AppendWithin(100, 0, contract.Message{Role: "assistant", Content: "x"}), ShouldBeNil)
		run.TruncateTo(mark)
		So(run.Length(), ShouldEqual, 1)
	})
}

func TestRedisStore_PersistRejectedOnWrongToken(t *testing.T) {
	Convey("错误 token 写文档被 Lua 拒绝", t, func() {
		_, client := miniClient(t)
		store := newRedisStoreWithLock(client, 8, time.Minute, 0).(*redisStore)
		ctx := context.Background()
		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		defer run.Release()
		err = store.persist(ctx, id, "not-the-token", sessionDoc{})
		So(err, ShouldEqual, contract.ErrInternal)
	})
}

func TestRedisStore_ClosedServer(t *testing.T) {
	Convey("miniredis 关闭后 Create 失败为 ErrInternal", t, func() {
		s, client := miniClient(t)
		store := newRedisStore(client, 8)
		s.Close()
		_, err := store.Create(context.Background())
		So(err, ShouldEqual, contract.ErrInternal)
	})
}

func TestRedisStore_LockExpireAllowsOtherStore(t *testing.T) {
	Convey("手动 DEL 锁 key 后第二 store 能 TryBeginRun", t, func() {
		s, client := miniClient(t)
		ctx := context.Background()
		a := newRedisStoreWithLock(client, 8, time.Minute, 0)
		b := newRedisStoreWithLock(client, 8, time.Minute, 0)
		id, err := a.Create(ctx)
		So(err, ShouldBeNil)
		run, err := a.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		s.Del(lockKey(id))
		run2, err := b.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		run.Release()
		run2.Release()
	})
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test ./framework/provider/agent/ -count=1 -run 'TestRedisStore_'`

Expected: FAIL，`newRedisStore` undefined。

- [ ] **Step 4: 实现 redis_store.go**

要点（完整实现按此结构，不要另发明 key 名）：

- `redisStore` 持有 `client`、`maxSessions`、`lockTTL`、`renewEvery`。
- `Create`：`SCARD` sessions set；达上限 `ErrSessionLimit`；UUID；`SET NX` 空 JSON `{"messages":[],"bytes":0}`；`SADD`；失败则 `DEL`+`SREM`。Redis 错误 → `ErrInternal`。
- `Open`：`GET`；`redis.Nil` → `ErrSessionNotFound`。
- `TryBeginRun`：先 `EXISTS` session key，没有 → `ErrSessionNotFound`；`SET lock NX PX lockTTL.Milliseconds()` 失败 → `ErrSessionBusy`；成功返回 `redisRunSession{store, id, token, doc}`。`renewEvery>0` 时起 goroutine：`time.Ticker` + `Eval luaRenew`，`Release` 时 `close(stop)` 再 `Eval luaUnlock`。
- `redisRunSession` 在内存里持有当前 `sessionDoc` 副本；`AppendWithin` 与内存相同的 `messageBytes` 计算，成功后 `persist`；`persist` 用 `luaPersist`，返回 0 → `ErrInternal`。
- `Release` 必须可重入安全（`sync.Once`），避免 double unlock。

`newRedisStore` 调用 `newRedisStoreWithLock(client, maxSessions, lockTTLDefault, renewDefault)`。

- [ ] **Step 5: 再跑 Redis 测试**

Run: `go test ./framework/provider/agent/ -count=1 -run 'TestRedisStore_'`

Expected: PASS。若 miniredis 不支持某 Lua，保持脚本语义、改用 `TxPipelined`/`WATCH` 等价实现，测试不得删。

- [ ] **Step 6: Commit**

跳过。

---

### Task 3: Provider 注入 Redis store

**Files:**
- Modify: `framework/provider/agent/provider.go`
- Modify: `framework/provider/agent/provider_test.go`
- Modify: `framework/provider/agent/service.go`（`NewHadeAgentService` 第 5 参为 `error` 或 `SessionStore`）

**Interfaces:**
- Consumes: `contract.RedisService.GetClient()`；Task 2 `newRedisStore`。
- Produces: `Params` 返回 `[]interface{}{llm, maxIter, limits, logger, storeOrErr}`，长度 5。

- [ ] **Step 1: 扩展 provider 测试**

`TestHadeAgentProvider_ParamsDefaults` 中 `ShouldHaveLength, 5`，无 Redis 时 `params[4]` 为 `nil`（`NewHadeAgentService` 则用内存 store）。

新增：

```go
type stubRedisService struct {
	client *redis.Client
	err    error
}

func (s *stubRedisService) GetClient(...contract.RedisOption) (*redis.Client, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

type stubRedisProvider struct {
	svc contract.RedisService
}

func (p *stubRedisProvider) Name() string { return contract.RedisKey }
func (p *stubRedisProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return p.svc, nil }
}
func (p *stubRedisProvider) Boot(framework.Container) error           { return nil }
func (p *stubRedisProvider) IsDefer() bool                            { return false }
func (p *stubRedisProvider) Params(framework.Container) []interface{} { return nil }
```

用例 1：miniredis client 注入 stub，`Params` 第 5 项是 `*redisStore`（或实现了 `SessionStore`），`NewHadeAgentService` 后 `agent.store` 能 `Create` 且数据出现在 miniredis。

用例 2：`stubRedisService{err: errors.New("dial")}`，`NewHadeAgentService(params...)` 返回非 nil error。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./framework/provider/agent/ -count=1 -run 'TestHadeAgentProvider_'`

Expected: FAIL（长度仍为 4，或第 5 项未注入）。

- [ ] **Step 3: 实现 Params / NewHadeAgentService**

`provider.go`：

```go
var store interface{}
if c.IsBind(contract.RedisKey) {
	redisService := c.MustMake(contract.RedisKey).(contract.RedisService)
	client, err := redisService.GetClient()
	if err != nil {
		store = err
	} else {
		store = newRedisStore(client, limits.MaxSessions)
	}
}
return []interface{}{llm, maxIter, limits, logger, store}
```

`NewHadeAgentService`：

```go
if len(params) > 4 && params[4] != nil {
	if err, ok := params[4].(error); ok {
		return nil, err
	}
	if store, ok := params[4].(SessionStore); ok {
		agent.store = store
	}
}
```

- [ ] **Step 4: 全包测试**

Run: `go test ./framework/provider/agent/ ./framework/agenthttp/ ./app/agent/ -count=1`

Expected: PASS。

- [ ] **Step 5: Commit**

跳过。

---

## Spec coverage

| Spec | Task |
|------|------|
| SessionStore / RunSession | 1 |
| 内存默认、现有 Agent 行为 | 1 |
| dangling settle | 1 |
| Redis key / Create / Open / 锁 / Lua / Busy 跨实例 | 2 |
| miniredis 8 类用例 | 2 |
| GetClient 一次；失败不退内存 | 3 |
| 不改 contract.Agent / 工具不进 Redis / 不用 Cache | 全局约束 |

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-30-agent-session-store-redis.md`. Two execution options:

**1. Subagent-Driven (recommended)** — 每个 Task 开一个新 subagent，任务之间我来 review

**2. Inline Execution** — 本会话按 executing-plans 逐项做，中间设检查点

Which approach?
