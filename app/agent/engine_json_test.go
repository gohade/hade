package agent

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSessionUsesSnakeCaseJSONWire(t *testing.T) {
	Convey("GET session 的原始 JSON 使用 snake_case，不出现 PascalCase 键", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Role:      "assistant",
					Content:   "calling",
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				},
				Finish: contract.FinishToolCalls,
			},
			{Message: contract.Message{Role: "assistant", Content: "bye"}, Finish: contract.FinishStop},
		}}
		mem := agprovider.NewMemoryAgent(script, 8)
		engine := newTestEngine(t, mem)

		id, err := mem.CreateSession(context.Background())
		So(err, ShouldBeNil)

		empty := performRequest(engine, http.MethodGet, "/sessions/"+id, nil)
		So(empty.Code, ShouldEqual, http.StatusOK)
		So(empty.Body.String(), ShouldEqual, `{"id":"`+id+`","messages":[]}`)

		run := performRequest(
			engine,
			http.MethodPost,
			"/sessions/"+id+"/messages",
			bytes.NewBufferString(`{"message":"hello"}`),
		)
		So(run.Code, ShouldEqual, http.StatusOK)

		get := performRequest(engine, http.MethodGet, "/sessions/"+id, nil)
		So(get.Code, ShouldEqual, http.StatusOK)
		body := get.Body.String()

		for _, want := range []string{
			`"id":"` + id + `"`,
			`"messages":[`,
			`"role":"user"`,
			`"content":"hello"`,
			`"tool_calls":[{"id":"c1","name":"echo","arguments":"{\"text\":\"hi\"}"}]`,
			`"tool_call_id":"c1"`,
		} {
			So(body, ShouldContainSubstring, want)
		}
		for _, unwanted := range []string{
			`"ID"`, `"Messages"`, `"Role"`, `"Content"`, `"ToolCalls"`, `"ToolCallID"`,
		} {
			So(body, ShouldNotContainSubstring, unwanted)
		}
		// 空的可空字段被 omitempty 省略：user 消息不带 tool_call_id/tool_calls。
		So(body, ShouldStartWith, `{"id":"`+id+`","messages":[{"role":"user","content":"hello"}`)
	})
}
