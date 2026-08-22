package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateSession_HitsSessionLimit(t *testing.T) {
	Convey("Session 数达到上限后返回 ErrSessionLimit", t, func() {
		a := NewMemoryAgentWithLimits(&fakeLLM{}, 8, Limits{MaxSessions: 2})
		first, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		second, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		So(first, ShouldNotEqual, second)

		_, err = a.CreateSession(context.Background())
		So(err, ShouldEqual, contract.ErrSessionLimit)

		// 已有 Session 仍然可用。
		_, err = a.GetSession(context.Background(), first)
		So(err, ShouldBeNil)
	})

	Convey("默认上限来自 contract 常量", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		So(a.limits.MaxSessions, ShouldEqual, contract.DefaultMaxSessions)
		So(a.limits.MaxMessageBytes, ShouldEqual, contract.DefaultMaxMessageBytes)
		So(a.limits.MaxHistoryBytes, ShouldEqual, contract.DefaultMaxHistoryBytes)
	})
}

func TestRun_RejectsOversizedUserMessage(t *testing.T) {
	Convey("单条 user 消息超限时不产生任何事件", t, func() {
		a := NewMemoryAgentWithLimits(&fakeLLM{}, 8, Limits{MaxMessageBytes: 16})
		id, _ := a.CreateSession(context.Background())
		events := make(chan contract.AgentEvent, 4)

		err := a.Run(context.Background(), id, strings.Repeat("x", 17), events)
		So(err, ShouldEqual, contract.ErrMessageTooLarge)
		So(len(events), ShouldEqual, 0)

		session := mustSession(a, id)
		So(session.Messages, ShouldHaveLength, 0)

		// 边界内的消息正常接受。
		So(a.Run(context.Background(), id, strings.Repeat("x", 16), events), ShouldBeNil)
	})
}

func TestRun_HistoryLimitRejectsOversizedLLMResponse(t *testing.T) {
	Convey("LLM 单次超大响应被拒绝，历史不越界", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{{
			Message: contract.Message{Role: "assistant", Content: strings.Repeat("y", 200)},
			Finish:  contract.FinishStop,
		}}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 16)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrHistoryLimit)

		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "history_limit")
		for _, event := range collected {
			So(event.Type, ShouldNotEqual, contract.EventFinal)
		}

		session := mustSession(a, id)
		So(session.Messages, ShouldHaveLength, 1)
		So(session.Messages[0].Role, ShouldEqual, "user")
		So(a.sess[id].usedBytes(), ShouldBeLessThanOrEqualTo, 70)
	})
}

func TestRun_HistoryLimitRollsBackDanglingToolCalls(t *testing.T) {
	Convey("工具结果写不下时回滚整组 tool_calls，历史既不越界也不残留悬空 call", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "echo", `{}`),
		}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(context.Context, string) (string, error) {
			return strings.Repeat("z", 200), nil
		})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 16)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrHistoryLimit)
		collect(events)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(session.Messages, ShouldHaveLength, 1)
		So(a.sess[id].usedBytes(), ShouldBeLessThanOrEqualTo, 70)
	})
}

func TestRun_CanceledSettlementStaysWithinHistoryLimit(t *testing.T) {
	Convey("补偿写入使用预留配额，取消后历史仍不越界", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "slow", `{}`),
		}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "unused", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
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
		So(a.sess[id].usedBytes(), ShouldBeLessThanOrEqualTo, 70)
	})
}
