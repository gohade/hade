package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	openAIToolCallBody = `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant",` +
		`"content":null,"tool_calls":[{"id":"c1","type":"function",` +
		`"function":{"name":"echo","arguments":"{\"text\":\"hi\"}"}}]}}]}`
	openAIStopBody = `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}]}`
)

func TestRun_ClosedLoopAgainstOpenAIServer(t *testing.T) {
	Convey("对接返回 200 的 OpenAI 兼容服务，ReAct 闭环完成且第二轮请求携带合法历史", t, func() {
		var (
			mu     sync.Mutex
			bodies [][]byte
		)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			mu.Lock()
			bodies = append(bodies, body)
			round := len(bodies)
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if round == 1 {
				_, _ = io.WriteString(w, openAIToolCallBody)
				return
			}
			_, _ = io.WriteString(w, openAIStopBody)
		}))
		defer server.Close()

		a := NewMemoryAgent(llmp.NewOpenAI(server.URL, "secret-key", "fixture-model"), 8)
		a.RegisterTool(contract.ToolSpec{
			Name: "echo",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"text": map[string]interface{}{"type": "string"}},
			},
		}, func(_ context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hello", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldBeNil)

		collected := collect(events)
		types := make([]string, len(collected))
		for i, event := range collected {
			types[i] = event.Type
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		So(collected[len(collected)-1].Data["content"], ShouldEqual, "done")

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{`{"text":"hi"}`})

		mu.Lock()
		defer mu.Unlock()
		So(bodies, ShouldHaveLength, 2)
		second := string(bodies[1])
		So(second, ShouldContainSubstring, `"model":"fixture-model"`)
		So(second, ShouldContainSubstring, `"tool_calls":[{"id":"c1","type":"function"`)
		So(second, ShouldContainSubstring, `"role":"tool"`)
		So(second, ShouldContainSubstring, `"tool_call_id":"c1"`)
	})
}
