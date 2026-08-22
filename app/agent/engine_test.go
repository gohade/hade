package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAgentEngineSessionAndMessagesSSE(t *testing.T) {
	Convey("创建、查询 session 并按序流式返回 ReAct 事件", t, func() {
		container := framework.NewHadeContainer()
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message:   contract.Message{Content: "t1"},
				ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				Finish:    contract.FinishToolCalls,
			},
			{Message: contract.Message{Content: "bye"}, Finish: contract.FinishStop},
		}}
		mem := agprovider.NewMemoryAgent(script, 8)
		So(container.Bind(&agentStub{agent: mem}), ShouldBeNil)

		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)

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
		So(data[2]["name"], ShouldEqual, "echo")
		So(data[2]["arguments"], ShouldResemble, map[string]interface{}{"text": "hi"})
		So(data[5]["content"], ShouldEqual, "bye")
		So(data[6], ShouldResemble, map[string]interface{}{})
	})
}

func TestAgentEngineMessagePreStreamErrors(t *testing.T) {
	Convey("不存在的 session 在升流前返回 404 JSON", t, func() {
		engine := newTestEngine(t, agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8))
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/nope/messages",
			bytes.NewBufferString(`{"message":"x"}`),
		)
		assertJSONError(response, http.StatusNotFound, contract.ErrSessionNotFound.Error())
	})

	Convey("空消息在升流前返回 400 JSON", t, func() {
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		engine := newTestEngine(t, mem)
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
		engine := newTestEngine(t, &preStreamErrorAgent{runErr: contract.ErrSessionBusy})
		response := performRequest(
			engine,
			http.MethodPost,
			"/sessions/busy/messages",
			bytes.NewBufferString(`{"message":"x"}`),
		)
		assertJSONError(response, http.StatusConflict, contract.ErrSessionBusy.Error())
	})
}

func newTestEngine(t *testing.T, agent contract.Agent) http.Handler {
	t.Helper()
	container := framework.NewHadeContainer()
	if err := container.Bind(&agentStub{agent: agent}); err != nil {
		t.Fatal(err)
	}
	engine, err := NewAgentEngine(container)
	if err != nil {
		t.Fatal(err)
	}
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

type agentStub struct {
	agent contract.Agent
}

func (stub *agentStub) Name() string { return contract.AgentKey }
func (stub *agentStub) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return stub.agent, nil }
}
func (stub *agentStub) Boot(framework.Container) error { return nil }
func (stub *agentStub) IsDefer() bool                  { return false }
func (stub *agentStub) Params(framework.Container) []interface{} {
	return nil
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
