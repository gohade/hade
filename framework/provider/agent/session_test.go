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
