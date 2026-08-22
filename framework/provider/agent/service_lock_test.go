package agent

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSession_NotBlockedByRunningRun(t *testing.T) {
	Convey("Run 长时间持有 runMu 时 GetSession 仍然快速返回", t, func() {
		llm := newCancelWaitLLM()
		a := NewMemoryAgent(llm, 8)
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "long run", events) }()

		select {
		case <-llm.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("LLM 未进入 Chat")
		}

		type snapshotResult struct {
			session contract.Session
			err     error
		}
		snapshot := make(chan snapshotResult, 1)
		go func() {
			session, err := a.GetSession(context.Background(), id)
			snapshot <- snapshotResult{session: session, err: err}
		}()
		select {
		case got := <-snapshot:
			So(got.err, ShouldBeNil)
			So(got.session.ID, ShouldEqual, id)
			// user 消息在 Run 开始时就已写入，读接口能看到。
			So(got.session.Messages, ShouldHaveLength, 1)
			So(got.session.Messages[0].Content, ShouldEqual, "long run")
		case <-time.After(time.Second):
			t.Fatal("GetSession 被正在运行的 Run 阻塞")
		}

		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)
	})
}

func TestMemoryAgent_ConcurrentReadWriteIsRaceFree(t *testing.T) {
	Convey("并发 Run / GetSession / RegisterTool / ListTools 无数据竞争", t, func() {
		script := &llmp.ScriptLLM{}
		for i := 0; i < 64; i++ {
			script.Responses = append(script.Responses, contract.ChatResponse{
				Message: contract.Message{Role: "assistant", Content: "final " + strconv.Itoa(i)},
				Finish:  contract.FinishStop,
			})
		}
		a := NewMemoryAgent(script, 8)

		ids := make([]string, 0, 8)
		for i := 0; i < 8; i++ {
			id, err := a.CreateSession(context.Background())
			So(err, ShouldBeNil)
			ids = append(ids, id)
		}

		// 断言统一放到 goroutine 之外，避免在并发上下文里调用 convey。
		var (
			mu      sync.Mutex
			failure []string
			wg      sync.WaitGroup
		)
		record := func(message string) {
			mu.Lock()
			failure = append(failure, message)
			mu.Unlock()
		}

		for _, id := range ids {
			sessionID := id
			wg.Add(3)
			go func() {
				defer wg.Done()
				events := make(chan contract.AgentEvent, 8)
				if err := a.Run(context.Background(), sessionID, "hi", events); err != nil &&
					err != contract.ErrSessionBusy {
					record("run: " + err.Error())
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					if _, err := a.GetSession(context.Background(), sessionID); err != nil {
						record("get: " + err.Error())
					}
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					a.RegisterTool(
						contract.ToolSpec{Name: "t" + strconv.Itoa(i%4)},
						func(context.Context, string) (string, error) { return "ok", nil },
					)
					_ = a.ListTools()
				}
			}()
		}
		wg.Wait()
		So(failure, ShouldBeEmpty)

		// 同名注册只覆盖不追加。
		So(a.ListTools(), ShouldHaveLength, 4)
		for _, id := range ids {
			assertHistoryValidForOpenAI(mustSession(a, id).Messages)
		}
	})
}
