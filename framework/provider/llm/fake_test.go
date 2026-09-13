package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScriptLLM_ReturnsScriptedToolCallThenStop(t *testing.T) {
	Convey("scripted chat", t, func() {
		req1 := contract.ChatRequest{
			Messages: []contract.Message{{Role: "user", Content: "hello"}},
		}
		req2 := contract.ChatRequest{
			Messages: []contract.Message{{Role: "user", Content: "follow up"}},
		}
		script := &ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Role:    "assistant",
					Content: "need echo",
					ToolCalls: []contract.ToolCall{{
						ID: "call_1", Name: "echo", Arguments: `{"text":"hi"}`,
					}},
				},
				Finish: contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "done"},
				Finish:  contract.FinishStop,
			},
		}}
		var svc contract.LLM = script
		r1, err := svc.Chat(context.Background(), req1)
		So(err, ShouldBeNil)
		So(r1.Finish, ShouldEqual, contract.FinishToolCalls)
		So(r1.Message.ToolCalls[0].Name, ShouldEqual, "echo")
		r2, err := svc.Chat(context.Background(), req2)
		So(err, ShouldBeNil)
		So(r2.Finish, ShouldEqual, contract.FinishStop)
		So(r2.Message.Content, ShouldEqual, "done")
		So(len(script.Calls), ShouldEqual, 2)
		So(script.Calls[0].Messages[0].Content, ShouldEqual, "hello")
		So(script.Calls[1].Messages[0].Content, ShouldEqual, "follow up")
	})
}

func TestScriptLLM_ReturnsScriptExhausted(t *testing.T) {
	Convey("script exhausted", t, func() {
		script := &ScriptLLM{Responses: []contract.ChatResponse{
			{Finish: contract.FinishStop},
		}}
		_, err := script.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldBeNil)
		_, err = script.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "llm script exhausted")
	})
}

func TestScriptLLM_ReturnsInjectedError(t *testing.T) {
	Convey("injected error", t, func() {
		wantErr := errors.New("injected failure")
		script := &ScriptLLM{
			Errs: []error{wantErr},
			Responses: []contract.ChatResponse{
				{Finish: contract.FinishStop},
			},
		}
		_, err := script.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldEqual, wantErr)
	})
}

func TestScriptLLM_RespectsCanceledContext(t *testing.T) {
	Convey("canceled context", t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		script := &ScriptLLM{Responses: []contract.ChatResponse{
			{Finish: contract.FinishStop},
		}}
		_, err := script.Chat(ctx, contract.ChatRequest{})
		So(err, ShouldEqual, context.Canceled)
		So(len(script.Calls), ShouldEqual, 0)
	})
}
