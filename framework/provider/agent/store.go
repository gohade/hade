package agent

import (
	"context"

	"github.com/gohade/hade/framework/contract"
)

// SessionStore 是 Agent Session 的持久化与互斥入口。
// 内存实现进程退出即丢失；Redis 实现可跨进程共享同一份历史。
// 工具表不在这里：RegisterTool 仍是进程内的。
type SessionStore interface {
	// Create 新建空 Session。
	Create(ctx context.Context) (id string, err error)
	// Open 只读快照，不抢 Run 锁，因此同一 Session 正在 Run 时仍可查询。
	// 不存在时返回 contract.ErrSessionNotFound。
	Open(ctx context.Context, id string) (messages []contract.Message, err error)
	// TryBeginRun 抢该 Session 的 Run 锁。调用方必须 Release（建议 defer）。
	// 不存在 → ErrSessionNotFound；已有一轮 Run 持锁 → ErrSessionBusy。
	TryBeginRun(ctx context.Context, id string) (RunSession, error)
}

// RunSession 是已持有 Run 锁的 Session 句柄，临界区覆盖整轮 ReAct。
// 写历史必须通过这里，以便 Redis 路径用锁 token 校验，避免僵尸进程覆盖。
type RunSession interface {
	// ID 返回 Session 标识。
	ID() string
	// Snapshot 返回消息深拷贝，调用方可长期持有与修改而不影响存储。
	Snapshot() []contract.Message
	// Length 返回当前消息条数，用于 TruncateTo 回滚。
	Length() int
	// UsedBytes 返回历史已占用的字节数（与 messageBytes 同一套统计）。
	UsedBytes() int
	// Append 追加消息。Redis 写失败时返回 contract.ErrInternal。
	Append(msgs ...contract.Message) error
	// TruncateTo 回滚到指定条数，用于撤销一组未闭环的 assistant.tool_calls。
	TruncateTo(n int)
	// Release 释放 Run 锁。必须成对调用；实现应可安全重复调用。
	Release()
}
