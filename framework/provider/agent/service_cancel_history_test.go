package agent

import (
	"context"
	"testing"
	"time"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

// cancelScenarioScript 第一轮返回一个 tool_call，第二轮直接收尾。
// 第二轮的 ChatRequest 就是"取消之后真正送给 LLM 的历史"。
func cancelScenarioScript() *llmp.ScriptLLM {
	return &llmp.ScriptLLM{Responses: []contract.ChatResponse{
		toolCallResponse("c1", "slow", `{"k":"v"}`),
		{Message: contract.Message{Role: "assistant", Content: "ok"}, Finish: contract.FinishStop},
	}}
}

func waitRunErr(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Run 取消后没有返回")
		return nil
	}
}

// assertSecondRunHistory 在同一个 Session 上再跑一轮，并断言送给 LLM 的历史合法。
func assertSecondRunHistory(t *testing.T, a *MemoryAgent, script *llmp.ScriptLLM, id string) {
	t.Helper()
	events := make(chan contract.AgentEvent, 32)
	done := make(chan error, 1)
	go func() {
		done <- a.Run(context.Background(), id, "again", events)
		close(events)
	}()
	So(waitRunErr(t, done), ShouldBeNil)
	collect(events)

	So(len(script.Calls), ShouldEqual, 2)
	assertHistoryValidForOpenAI(script.Calls[1].Messages)
	assertHistoryValidForOpenAI(mustSession(a, id).Messages)
}

func mustSession(a *MemoryAgent, id string) contract.Session {
	session, err := a.GetSession(context.Background(), id)
	So(err, ShouldBeNil)
	return session
}

func TestRun_CancelBeforeActionSentClosesToolCall(t *testing.T) {
	Convey("取消发生在 action 事件发送前：pending 的 tool_call 被补 canceled", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "should not run", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		// 不带缓冲：读到 thought 后停止读取，Run 必然卡在 action 的 send 上。
		events := make(chan contract.AgentEvent)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "hi", events) }()

		So((<-events).Type, ShouldEqual, contract.EventSession)
		So((<-events).Type, ShouldEqual, contract.EventThought)
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{settleReasonCanceled})

		assertSecondRunHistory(t, a, script, id)
	})
}

func TestRun_CancelBeforeObservationSentKeepsRealResult(t *testing.T) {
	Convey("取消发生在 observation 事件发送前：保留真实结果且不重复补偿", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "real result", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "hi", events) }()

		So((<-events).Type, ShouldEqual, contract.EventSession)
		So((<-events).Type, ShouldEqual, contract.EventThought)
		So((<-events).Type, ShouldEqual, contract.EventAction)
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		// 真实结果先落历史再推事件，所以这里必须是工具结果而不是 canceled。
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{"real result"})

		assertSecondRunHistory(t, a, script, id)
	})
}

func TestRun_CancelDuringToolExecutionKeepsHistoryValid(t *testing.T) {
	Convey("取消发生在工具执行中途：历史依然闭环", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		entered := make(chan struct{})
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(ctx context.Context, _ string) (string, error) {
			close(entered)
			<-ctx.Done()
			return "midway", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(ctx, id, "hi", events)
			close(events)
		}()

		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("工具未被调用")
		}
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)
		collect(events)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		replies := toolReplies(session.Messages)["c1"]
		So(replies, ShouldHaveLength, 1)
		// 工具在取消后仍然返回了结果，这个结果或补偿值都必须闭环该 call。
		So(replies[0] == "midway" || replies[0] == settleReasonCanceled, ShouldBeTrue)

		assertSecondRunHistory(t, a, script, id)
	})
}
