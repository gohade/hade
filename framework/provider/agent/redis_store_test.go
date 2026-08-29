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
