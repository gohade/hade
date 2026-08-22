package agent

import (
	"context"
	"testing"
	"time"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

// toolReplies 汇总历史里每个 tool_call_id 收到的回复内容。
func toolReplies(messages []contract.Message) map[string][]string {
	replies := map[string][]string{}
	for _, message := range messages {
		if message.Role == "tool" {
			replies[message.ToolCallID] = append(replies[message.ToolCallID], message.Content)
		}
	}
	return replies
}

// assertHistoryValidForOpenAI 断言历史满足 OpenAI 约束：
// 每个 assistant.tool_calls 里的 call id 都有且只有一条 tool 回复。
func assertHistoryValidForOpenAI(messages []contract.Message) {
	replies := toolReplies(messages)
	calls := 0
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		for _, call := range message.ToolCalls {
			calls++
			So(replies[call.ID], ShouldHaveLength, 1)
		}
	}
	// 不允许出现没有对应 assistant.tool_call 的孤立 tool 消息。
	So(len(replies), ShouldEqual, calls)
}

func TestRun_ToolPanicBecomesObservationAndLoopReachesFinal(t *testing.T) {
	Convey("第三方工具 panic 转成 error observation，循环仍然走到 final", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "boom", `{}`),
			{Message: contract.Message{Role: "assistant", Content: "recovered"}, Finish: contract.FinishStop},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "boom"}, func(context.Context, string) (string, error) {
			panic("tool exploded")
		})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 卡死")
		}
		So(runErr, ShouldBeNil)

		collected := collect(events)
		types := make([]string, len(collected))
		var observation string
		for i, event := range collected {
			types[i] = event.Type
			if event.Type == contract.EventObservation {
				observation, _ = event.Data["content"].(string)
			}
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		So(observation, ShouldContainSubstring, "panicked")
		So(observation, ShouldContainSubstring, "tool exploded")

		session, _ := a.GetSession(context.Background(), id)
		assertHistoryValidForOpenAI(session.Messages)
	})
}

func TestRun_PanicInsideRunSettlesPendingToolCalls(t *testing.T) {
	Convey("Run 自身 panic 时补齐未闭环 tool_call 并返回 ErrInternal", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{{
			Message: contract.Message{
				Role:    "assistant",
				Content: "two calls",
				ToolCalls: []contract.ToolCall{
					{ID: "c1", Name: "closer", Arguments: `{}`},
					{ID: "c2", Name: "closer", Arguments: `{}`},
				},
			},
			Finish: contract.FinishToolCalls,
		}}}
		a := NewMemoryAgent(script, 8)
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 32)
		// 模拟调用方提前关闭事件 channel：后续 send 会 panic。
		a.RegisterTool(contract.ToolSpec{Name: "closer"}, func(context.Context, string) (string, error) {
			close(events)
			return "ok", nil
		})

		done := make(chan error, 1)
		go func() { done <- a.Run(context.Background(), id, "go", events) }()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 未从 panic 中恢复")
		}
		So(runErr, ShouldEqual, contract.ErrInternal)

		session, _ := a.GetSession(context.Background(), id)
		assertHistoryValidForOpenAI(session.Messages)
		replies := toolReplies(session.Messages)
		So(replies["c1"][0], ShouldEqual, "ok")
		So(replies["c2"][0], ShouldEqual, settleReasonInternal)
	})
}

func TestRun_PanicInLLMReturnsInternal(t *testing.T) {
	Convey("LLM 实现 panic 时 Run 返回 ErrInternal 且发 internal 事件", t, func() {
		a := NewMemoryAgent(&panicLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 未从 panic 中恢复")
		}
		So(runErr, ShouldEqual, contract.ErrInternal)

		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "internal")
	})
}

type panicLLM struct{}

func (*panicLLM) Chat(context.Context, contract.ChatRequest) (contract.ChatResponse, error) {
	panic("llm exploded")
}
