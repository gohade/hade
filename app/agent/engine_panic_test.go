package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

// panicAgent 模拟第三方 contract.Agent 实现的 Run panic。
type panicAgent struct {
	emitFirst bool
}

func (*panicAgent) CreateSession(context.Context) (string, error) { return "s1", nil }
func (*panicAgent) GetSession(context.Context, string) (contract.Session, error) {
	return contract.Session{ID: "s1"}, nil
}
func (*panicAgent) RegisterTool(contract.ToolSpec, contract.ToolHandler) {}
func (*panicAgent) ListTools() []contract.ToolSpec                       { return nil }
func (a *panicAgent) Run(
	_ context.Context,
	sessionID, _ string,
	events chan<- contract.AgentEvent,
) error {
	if a.emitFirst {
		events <- contract.AgentEvent{
			Type: contract.EventSession,
			Data: map[string]interface{}{"session_id": sessionID},
		}
	}
	panic("third party agent exploded")
}

func TestMessagesRecoversPreStreamAgentPanic(t *testing.T) {
	Convey("升流前的 Agent.Run panic 返回 500 JSON，不升 SSE，进程不崩", t, func() {
		engine := newTestEngine(t, &panicAgent{})
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/s1/messages",
			bytes.NewBufferString(`{"message":"boom"}`),
		)
		assertJSONError(response, http.StatusInternalServerError, contract.ErrInternal.Error())

		// 进程存活，后续请求照常处理。
		So(performRequest(engine, http.MethodGet, "/sessions/s1", nil).Code, ShouldEqual, http.StatusOK)
	})
}

func TestMessagesRecoversStreamedAgentPanic(t *testing.T) {
	Convey("已升流的 Agent.Run panic 补唯一 internal error 后仍发 done", t, func() {
		engine := newTestEngine(t, &panicAgent{emitFirst: true})
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/s1/messages",
			bytes.NewBufferString(`{"message":"boom"}`),
		)
		So(response.Code, ShouldEqual, http.StatusOK)
		So(response.Header().Get("Content-Type"), ShouldStartWith, "text/event-stream")

		frames, err := parseSSEFrames(response.Body.String())
		So(err, ShouldBeNil)
		types := make([]string, len(frames))
		for i, frame := range frames {
			types[i] = frame.Event
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventError, contract.EventDone,
		})

		var errorData map[string]interface{}
		So(json.Unmarshal(frames[1].Data, &errorData), ShouldBeNil)
		So(errorData["code"], ShouldEqual, "internal")
		So(frames[len(frames)-1].ID, ShouldEqual, len(frames))

		So(performRequest(engine, http.MethodGet, "/sessions/s1", nil).Code, ShouldEqual, http.StatusOK)
	})
}

// silentFailAgent 已升流但只返回错误、不发 error 事件，模拟第三方实现的疏漏。
type silentFailAgent struct {
	runErr error
}

func (*silentFailAgent) CreateSession(context.Context) (string, error) { return "s1", nil }
func (*silentFailAgent) GetSession(context.Context, string) (contract.Session, error) {
	return contract.Session{ID: "s1"}, nil
}
func (*silentFailAgent) RegisterTool(contract.ToolSpec, contract.ToolHandler) {}
func (*silentFailAgent) ListTools() []contract.ToolSpec                       { return nil }
func (a *silentFailAgent) Run(
	_ context.Context,
	sessionID, _ string,
	events chan<- contract.AgentEvent,
) error {
	events <- contract.AgentEvent{
		Type: contract.EventSession,
		Data: map[string]interface{}{"session_id": sessionID},
	}
	return a.runErr
}

func TestMessagesSupplementsErrorEventForAnyRunFailure(t *testing.T) {
	cases := []struct {
		name     string
		runErr   error
		wantCode string
	}{
		{name: "max_iterations", runErr: contract.ErrMaxIterations, wantCode: "max_iterations"},
		{name: "canceled", runErr: contract.ErrCanceled, wantCode: "canceled"},
		{name: "llm_failed", runErr: contract.ErrLLMFailed, wantCode: "llm_failed"},
		{name: "history_limit", runErr: contract.ErrHistoryLimit, wantCode: "history_limit"},
		{name: "unknown", runErr: errors.New("something else"), wantCode: "internal"},
	}
	for _, testCase := range cases {
		current := testCase
		Convey("已升流后 Run 返回 "+current.name+" 但没发 error 事件时由 handler 补一条", t, func() {
			engine := newTestEngine(t, &silentFailAgent{runErr: current.runErr})
			response := performRequest(
				engine,
				http.MethodPost,
				"/sessions/s1/messages",
				bytes.NewBufferString(`{"message":"go"}`),
			)
			So(response.Code, ShouldEqual, http.StatusOK)

			frames, err := parseSSEFrames(response.Body.String())
			So(err, ShouldBeNil)
			types := make([]string, len(frames))
			for i, frame := range frames {
				types[i] = frame.Event
			}
			So(types, ShouldResemble, []string{
				contract.EventSession, contract.EventError, contract.EventDone,
			})

			var errorData map[string]interface{}
			So(json.Unmarshal(frames[1].Data, &errorData), ShouldBeNil)
			So(errorData["code"], ShouldEqual, current.wantCode)
			// 未知错误的原始文案不外泄。
			So(response.Body.String(), ShouldNotContainSubstring, "something else")
		})
	}
}

func TestMessagesDoesNotDuplicateExistingErrorEvent(t *testing.T) {
	Convey("Run 已经发过 error 事件时 handler 不重复补", t, func() {
		script := &llmp.ScriptLLM{Errs: []error{errors.New("upstream down")}}
		mem := agprovider.NewMemoryAgent(script, 8)
		engine := newTestEngine(t, mem)
		id, err := mem.CreateSession(context.Background())
		So(err, ShouldBeNil)

		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/"+id+"/messages",
			bytes.NewBufferString(`{"message":"go"}`),
		)
		So(response.Code, ShouldEqual, http.StatusOK)

		frames, err := parseSSEFrames(response.Body.String())
		So(err, ShouldBeNil)
		errorFrames := 0
		for _, frame := range frames {
			if frame.Event == contract.EventError {
				errorFrames++
			}
		}
		So(errorFrames, ShouldEqual, 1)
		So(frames[len(frames)-1].Event, ShouldEqual, contract.EventDone)
	})
}

func TestMessagesToolPanicStillReachesFinal(t *testing.T) {
	Convey("工具 panic 只变成一条 observation，SSE 仍然走到 final + done", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Content:   "call boom",
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "boom", Arguments: `{}`}},
				},
				Finish: contract.FinishToolCalls,
			},
			{Message: contract.Message{Content: "survived"}, Finish: contract.FinishStop},
		}}
		mem := agprovider.NewMemoryAgent(script, 8)
		mem.RegisterTool(contract.ToolSpec{Name: "boom"}, func(context.Context, string) (string, error) {
			panic("tool exploded")
		})
		engine := newTestEngine(t, mem)

		id, err := mem.CreateSession(context.Background())
		So(err, ShouldBeNil)
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/"+id+"/messages",
			bytes.NewBufferString(`{"message":"hello"}`),
		)
		So(response.Code, ShouldEqual, http.StatusOK)

		frames, err := parseSSEFrames(response.Body.String())
		So(err, ShouldBeNil)
		types := make([]string, len(frames))
		for i, frame := range frames {
			types[i] = frame.Event
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
			contract.EventDone,
		})

		var observation map[string]interface{}
		So(json.Unmarshal(frames[3].Data, &observation), ShouldBeNil)
		So(observation["name"], ShouldEqual, "boom")
		So(observation["content"], ShouldContainSubstring, "panicked")

		var final map[string]interface{}
		So(json.Unmarshal(frames[5].Data, &final), ShouldBeNil)
		So(final["content"], ShouldEqual, "survived")
	})
}
