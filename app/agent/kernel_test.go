package agent

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

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

func toolNames(agent contract.Agent) map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range agent.ListTools() {
		out[spec.Name] = struct{}{}
	}
	return out
}

type countingAgentProvider struct {
	count int32
	agent contract.Agent
}

func (p *countingAgentProvider) Name() string { return contract.AgentKey }
func (p *countingAgentProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) {
		atomic.AddInt32(&p.count, 1)
		return p.agent, nil
	}
}
func (p *countingAgentProvider) Boot(framework.Container) error           { return nil }
func (p *countingAgentProvider) IsDefer() bool                            { return true }
func (p *countingAgentProvider) Params(framework.Container) []interface{} { return nil }
func (p *countingAgentProvider) instantiations() int32                    { return atomic.LoadInt32(&p.count) }

type stubORMProvider struct{}

func (p *stubORMProvider) Name() string { return contract.ORMKey }
func (p *stubORMProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return struct{}{}, nil }
}
func (p *stubORMProvider) Boot(framework.Container) error           { return nil }
func (p *stubORMProvider) IsDefer() bool                            { return true }
func (p *stubORMProvider) Params(framework.Container) []interface{} { return nil }

func TestNewAgentEngineRegistersToolsOnConstruct(t *testing.T) {
	Convey("构造 Engine 时实例化 Agent 并注册 echo/time，后续请求不再实例化", t, func() {
		container := framework.NewHadeContainer()
		provider := &countingAgentProvider{agent: agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)}
		So(container.Bind(provider), ShouldBeNil)

		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)
		So(provider.instantiations(), ShouldEqual, 1)
		So(provider.agent.ListTools(), ShouldHaveLength, 2)

		create := performRequest(engine, http.MethodPost, "/sessions", nil)
		So(create.Code, ShouldEqual, http.StatusCreated)
		So(provider.instantiations(), ShouldEqual, 1)

		for i := 0; i < 3; i++ {
			So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		}
		So(performRequest(engine, http.MethodGet, "/sessions/missing", nil).Code, ShouldEqual, http.StatusNotFound)
		So(provider.instantiations(), ShouldEqual, 1)
		So(provider.agent.ListTools(), ShouldHaveLength, 2)
	})
}

func TestNewAgentEngineSucceedsWithoutAgentBinding(t *testing.T) {
	Convey("容器里没有 Agent 时 Engine 仍能构造，请求返回 500 JSON 且可重试", t, func() {
		container := framework.NewHadeContainer()
		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)

		failed := performRequest(engine, http.MethodPost, "/sessions", nil)
		assertJSONError(failed, http.StatusInternalServerError, "agent service unavailable")
		So(failed.Body.String(), ShouldNotContainSubstring, contract.AgentKey)

		provider := &countingAgentProvider{agent: agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)}
		So(container.Bind(provider), ShouldBeNil)
		So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		So(provider.instantiations(), ShouldEqual, 1)
	})
}

func TestNewAgentEngineDoesNotTouchRealProviders(t *testing.T) {
	Convey("只绑定真实 Agent Provider（缺 LLM）：构造成功且不 panic", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&agprovider.HadeAgentProvider{}), ShouldBeNil)
		So(func() {
			engine, err := NewAgentEngine(container)
			So(err, ShouldBeNil)
			So(engine, ShouldNotBeNil)
		}, ShouldNotPanic)
	})
}

func TestRegisterExampleTools_SkipsUserToolsWithoutORM(t *testing.T) {
	Convey("未绑定 ORM 时只有 echo 和 time", t, func() {
		container := framework.NewHadeContainer()
		mem := agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "echo")
		So(names, ShouldContainKey, "time")
		So(names, ShouldNotContainKey, "create_user")
		So(names, ShouldNotContainKey, "get_user")
		So(names, ShouldNotContainKey, "list_users")
	})
}

func TestRegisterExampleTools_RegistersUserToolsWhenORMBound(t *testing.T) {
	Convey("绑定 ORM 关键字后注册三个 User 工具", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&stubORMProvider{}), ShouldBeNil)
		mem := agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "create_user")
		So(names, ShouldContainKey, "get_user")
		So(names, ShouldContainKey, "list_users")
	})
}
