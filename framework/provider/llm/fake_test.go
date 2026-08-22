package llm

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScriptLLM_ReturnsScriptedToolCallThenStop(t *testing.T) {
	Convey("scripted chat", t, func() {
		script := &ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{Role: "assistant", Content: "need echo"},
				ToolCalls: []contract.ToolCall{{
					ID: "call_1", Name: "echo", Arguments: `{"text":"hi"}`,
				}},
				Finish: contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "done"},
				Finish:  contract.FinishStop,
			},
		}}
		var svc contract.LLM = script
		r1, err := svc.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldBeNil)
		So(r1.Finish, ShouldEqual, contract.FinishToolCalls)
		So(r1.ToolCalls[0].Name, ShouldEqual, "echo")
		r2, err := svc.Chat(context.Background(), contract.ChatRequest{})
		So(err, ShouldBeNil)
		So(r2.Finish, ShouldEqual, contract.FinishStop)
		So(r2.Message.Content, ShouldEqual, "done")
	})
}
