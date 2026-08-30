package agent

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryStore_CreateOpenBusy(t *testing.T) {
	Convey("内存 store：创建、只读、Busy", t, func() {
		store := newMemoryStore()
		ctx := context.Background()

		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		msgs, err := store.Open(ctx, id)
		So(err, ShouldBeNil)
		So(msgs, ShouldBeEmpty)

		_, err = store.Create(ctx)
		So(err, ShouldBeNil)

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

func TestMemoryStore_AppendAndTruncate(t *testing.T) {
	Convey("内存 store：追加与 truncate", t, func() {
		store := newMemoryStore()
		ctx := context.Background()
		id, err := store.Create(ctx)
		So(err, ShouldBeNil)
		run, err := store.TryBeginRun(ctx, id)
		So(err, ShouldBeNil)
		defer run.Release()

		So(run.Append(contract.Message{Role: "user", Content: "hi"}), ShouldBeNil)
		So(run.Length(), ShouldEqual, 1)
		mark := run.Length()
		So(run.Append(contract.Message{Role: "assistant", Content: "there"}), ShouldBeNil)
		run.TruncateTo(mark)
		So(run.Length(), ShouldEqual, 1)
		So(run.Snapshot()[0].Content, ShouldEqual, "hi")
	})
}
