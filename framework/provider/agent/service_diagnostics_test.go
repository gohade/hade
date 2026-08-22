package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRun_PanicIsLoggedWithValueAndStack(t *testing.T) {
	Convey("Run recover 把 panic value 与调用栈写进诊断输出，但不外泄给客户端", t, func() {
		a := NewMemoryAgent(&panicLLM{}, 8)
		var diagnostics bytes.Buffer
		a.diagnostics = &diagnostics

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrInternal)

		logged := diagnostics.String()
		So(logged, ShouldContainSubstring, "run panic")
		So(logged, ShouldContainSubstring, id)
		// panic value
		So(logged, ShouldContainSubstring, "llm exploded")
		// debug.Stack：goroutine 头 + 本包的栈帧
		So(logged, ShouldContainSubstring, "goroutine")
		So(logged, ShouldContainSubstring, "framework/provider/agent")
		So(logged, ShouldContainSubstring, "MemoryAgent).Run")

		// 事件流只给出 sentinel，panic 细节一律不进事件。
		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "internal")
		So(fmt.Sprint(last.Data), ShouldNotContainSubstring, "llm exploded")
		So(fmt.Sprint(last.Data), ShouldNotContainSubstring, "goroutine")
	})

	Convey("未注入 writer 时默认落到 os.Stderr 且不 panic", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		So(a.diagnostics, ShouldNotBeNil)
		a.diagnostics = nil
		So(func() { a.logDiagnostic("[agent] probe %d\n", 1) }, ShouldNotPanic)
	})
}

func TestGetSession_TruncatesToolCallArguments(t *testing.T) {
	Convey("GetSession 同时截断 Content 与 tool arguments", t, func() {
		hugeArguments := `{"text":"` + strings.Repeat("z", contract.ContentMaxBytes*2) + `"}`
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Role:      "assistant",
					Content:   strings.Repeat("c", contract.ContentMaxBytes*2),
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: hugeArguments}},
				},
				Finish: contract.FinishToolCalls,
			},
			{Message: contract.Message{Role: "assistant", Content: "done"}, Finish: contract.FinishStop},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(context.Context, string) (string, error) {
			return "ok", nil
		})

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldBeNil)
		collect(events)

		session := mustSession(a, id)
		var assistantCalls []contract.ToolCall
		for _, message := range session.Messages {
			So(len(message.Content), ShouldBeLessThanOrEqualTo, contract.ContentMaxBytes)
			if message.Role == "assistant" && len(message.ToolCalls) > 0 {
				assistantCalls = message.ToolCalls
			}
		}
		So(assistantCalls, ShouldHaveLength, 1)
		So(len(assistantCalls[0].Arguments), ShouldEqual, contract.ContentMaxBytes)
		So(len(hugeArguments), ShouldBeGreaterThan, contract.ContentMaxBytes)

		// 快照被截断，内部历史仍是完整值（下一轮要原样送回 LLM）。
		So(len(a.sess[id].snapshot()[1].ToolCalls[0].Arguments), ShouldEqual, len(hugeArguments))
	})
}
