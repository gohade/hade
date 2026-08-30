package agenthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

type agentStub struct {
	agent contract.Agent
}

func (stub *agentStub) Name() string { return contract.AgentKey }
func (stub *agentStub) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return stub.agent, nil }
}
func (stub *agentStub) Boot(framework.Container) error           { return nil }
func (stub *agentStub) IsDefer() bool                            { return false }
func (stub *agentStub) Params(framework.Container) []interface{} { return nil }

func registerEcho(mem *agprovider.AgentRuntime) {
	mem.RegisterTool(contract.ToolSpec{Name: "echo"}, func(_ context.Context, argsJSON string) (string, error) {
		var args struct {
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) == nil && args.Text != "" {
			return args.Text, nil
		}
		return argsJSON, nil
	})
}

func newProtocolEngine(t *testing.T, agent contract.Agent) http.Handler {
	t.Helper()
	container := framework.NewHadeContainer()
	if err := container.Bind(&agentStub{agent: agent}); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetContainer(container)
	Mount(engine)
	return engine
}

func performRequest(handler http.Handler, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, body)
		request.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

func assertJSONError(response *httptest.ResponseRecorder, status int, message string) {
	So(response.Code, ShouldEqual, status)
	So(response.Header().Get("Content-Type"), ShouldStartWith, "application/json")
	So(response.Body.String(), ShouldEqual, `{"error":"`+message+`"}`)
	So(response.Header().Get("Content-Type"), ShouldNotStartWith, "text/event-stream")
}

type sseFrame struct {
	ID    int
	Event string
	Data  json.RawMessage
}

func parseSSEFrames(body string) ([]sseFrame, error) {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	blocks := strings.Split(strings.TrimSpace(normalized), "\n\n")
	frames := make([]sseFrame, 0, len(blocks))
	for _, block := range blocks {
		var frame sseFrame
		var dataLines []string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "id:"):
				id, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "id:")))
				if err != nil {
					return nil, fmt.Errorf("invalid SSE id %q: %w", line, err)
				}
				frame.ID = id
			case strings.HasPrefix(line, "event:"):
				frame.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if frame.ID == 0 || frame.Event == "" || len(dataLines) == 0 {
			return nil, fmt.Errorf("incomplete SSE frame %q", block)
		}
		frame.Data = json.RawMessage(strings.Join(dataLines, "\n"))
		frames = append(frames, frame)
	}
	return frames, nil
}

type preStreamErrorAgent struct {
	runErr error
}

func (*preStreamErrorAgent) CreateSession(context.Context) (string, error) {
	return "busy", nil
}
func (*preStreamErrorAgent) GetSession(context.Context, string) (contract.Session, error) {
	return contract.Session{ID: "busy"}, nil
}
func (*preStreamErrorAgent) RegisterTool(contract.ToolSpec, contract.ToolHandler) {}
func (*preStreamErrorAgent) ListTools() []contract.ToolSpec                       { return nil }
func (agent *preStreamErrorAgent) Run(context.Context, string, string, chan<- contract.AgentEvent) error {
	return agent.runErr
}

func TestMount_CreateSession201(t *testing.T) {
	Convey("Mount 后 POST /sessions 返回 201 和 id", t, func() {
		mem := agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)
		engine := newProtocolEngine(t, mem)
		resp := performRequest(engine, http.MethodPost, "/sessions", nil)
		So(resp.Code, ShouldEqual, http.StatusCreated)
		So(resp.Body.String(), ShouldContainSubstring, `"id":`)
	})
}

func TestAgentEngineSessionAndMessagesSSE(t *testing.T) {
	Convey("创建、查询 session 并按序流式返回 ReAct 事件", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Content:   "t1",
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				},
				Finish: contract.FinishToolCalls,
			},
			{Message: contract.Message{Content: "bye"}, Finish: contract.FinishStop},
		}}
		mem := agprovider.NewAgentRuntime(script, 8)
		registerEcho(mem)
		engine := newProtocolEngine(t, mem)

		create := performRequest(engine, http.MethodPost, "/sessions", nil)
		So(create.Code, ShouldEqual, http.StatusCreated)
		var created struct {
			ID string `json:"id"`
		}
		So(json.Unmarshal(create.Body.Bytes(), &created), ShouldBeNil)
		So(created.ID, ShouldNotBeBlank)

		get := performRequest(engine, http.MethodGet, "/sessions/"+created.ID, nil)
		So(get.Code, ShouldEqual, http.StatusOK)
		var session contract.Session
		So(json.Unmarshal(get.Body.Bytes(), &session), ShouldBeNil)
		So(session.ID, ShouldEqual, created.ID)

		messages := performRequest(
			engine,
			http.MethodPost,
			"/sessions/"+created.ID+"/messages",
			bytes.NewBufferString(`{"message":"hello"}`),
		)
		So(messages.Code, ShouldEqual, http.StatusOK)
		So(messages.Header().Get("Content-Type"), ShouldStartWith, "text/event-stream")

		frames, err := parseSSEFrames(messages.Body.String())
		So(err, ShouldBeNil)
		So(frames, ShouldHaveLength, 7)

		eventTypes := make([]string, len(frames))
		ids := make([]int, len(frames))
		data := make([]map[string]interface{}, len(frames))
		for i, frame := range frames {
			eventTypes[i] = frame.Event
			ids[i] = frame.ID
			So(json.Unmarshal(frame.Data, &data[i]), ShouldBeNil)
		}
		So(eventTypes, ShouldResemble, []string{
			contract.EventSession,
			contract.EventThought,
			contract.EventAction,
			contract.EventObservation,
			contract.EventThought,
			contract.EventFinal,
			contract.EventDone,
		})
		So(ids, ShouldResemble, []int{1, 2, 3, 4, 5, 6, 7})

		So(data[0]["session_id"], ShouldEqual, created.ID)
		So(data[1]["content"], ShouldEqual, "t1")
		So(data[2]["name"], ShouldEqual, "echo")
		So(data[2]["arguments"], ShouldResemble, map[string]interface{}{"text": "hi"})
		So(data[3]["name"], ShouldEqual, "echo")
		So(data[3]["content"], ShouldEqual, "hi")
		So(data[4]["content"], ShouldEqual, "bye")
		So(data[5]["content"], ShouldEqual, "bye")
		So(data[6], ShouldResemble, map[string]interface{}{})
	})
}

func TestAgentEngineMessagePreStreamErrors(t *testing.T) {
	Convey("不存在的 session 在升流前返回 404 JSON", t, func() {
		engine := newProtocolEngine(t, agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8))
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/nope/messages",
			bytes.NewBufferString(`{"message":"x"}`),
		)
		assertJSONError(response, http.StatusNotFound, contract.ErrSessionNotFound.Error())
	})

	Convey("空消息在升流前返回 400 JSON", t, func() {
		mem := agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)
		engine := newProtocolEngine(t, mem)
		id, err := mem.CreateSession(context.Background())
		So(err, ShouldBeNil)

		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/"+id+"/messages",
			bytes.NewBufferString(`{"message":""}`),
		)
		assertJSONError(response, http.StatusBadRequest, contract.ErrEmptyMessage.Error())
	})

	Convey("占用中的 session 在升流前返回 409 JSON", t, func() {
		engine := newProtocolEngine(t, &preStreamErrorAgent{runErr: contract.ErrSessionBusy})
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/busy/messages",
			bytes.NewBufferString(`{"message":"x"}`),
		)
		assertJSONError(response, http.StatusConflict, contract.ErrSessionBusy.Error())
	})
}

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
		mem := agprovider.NewAgentRuntime(script, 8)
		registerEcho(mem)
		engine := newProtocolEngine(t, mem)

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
		So(body, ShouldStartWith, `{"id":"`+id+`","messages":[{"role":"user","content":"hello"}`)
	})
}

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
		engine := newProtocolEngine(t, &panicAgent{})
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/s1/messages",
			bytes.NewBufferString(`{"message":"boom"}`),
		)
		assertJSONError(response, http.StatusInternalServerError, contract.ErrInternal.Error())
		So(performRequest(engine, http.MethodGet, "/sessions/s1", nil).Code, ShouldEqual, http.StatusOK)
	})
}

func TestMessagesRecoversStreamedAgentPanic(t *testing.T) {
	Convey("已升流的 Agent.Run panic 补唯一 internal error 后仍发 done", t, func() {
		engine := newProtocolEngine(t, &panicAgent{emitFirst: true})
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
		{name: "unknown", runErr: errors.New("something else"), wantCode: "internal"},
	}
	for _, testCase := range cases {
		current := testCase
		Convey("已升流后 Run 返回 "+current.name+" 但没发 error 事件时由 handler 补一条", t, func() {
			engine := newProtocolEngine(t, &silentFailAgent{runErr: current.runErr})
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
			So(response.Body.String(), ShouldNotContainSubstring, "something else")
		})
	}
}

func TestMessagesDoesNotDuplicateExistingErrorEvent(t *testing.T) {
	Convey("Run 已经发过 error 事件时 handler 不重复补", t, func() {
		script := &llmp.ScriptLLM{Errs: []error{errors.New("upstream down")}}
		mem := agprovider.NewAgentRuntime(script, 8)
		engine := newProtocolEngine(t, mem)
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
		mem := agprovider.NewAgentRuntime(script, 8)
		mem.RegisterTool(contract.ToolSpec{Name: "boom"}, func(context.Context, string) (string, error) {
			panic("tool exploded")
		})
		engine := newProtocolEngine(t, mem)

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

func TestMessagesRejectsOversizedBody(t *testing.T) {
	Convey("请求体超过 64KiB 时返回 413 JSON 且不升流", t, func() {
		mem := agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)
		engine := newProtocolEngine(t, mem)
		id, err := mem.CreateSession(context.Background())
		So(err, ShouldBeNil)

		body := bytes.NewBufferString(`{"message":"`)
		body.Write(bytes.Repeat([]byte("x"), contract.RequestBodyMaxBytes+1024))
		body.WriteString(`"}`)

		response := performRequest(engine, http.MethodPost, "/sessions/"+id+"/messages", body)
		So(response.Code, ShouldEqual, http.StatusRequestEntityTooLarge)
		So(response.Header().Get("Content-Type"), ShouldStartWith, "application/json")
		So(response.Header().Get("Content-Type"), ShouldNotStartWith, "text/event-stream")
	})
}
