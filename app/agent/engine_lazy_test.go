package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

// countingAgentProvider 记录 Agent 被实例化的次数，用于验证 defer 语义。
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

func TestNewAgentEngineDoesNotInstantiateAgent(t *testing.T) {
	Convey("构造 Engine 不实例化 Agent，首个请求才实例化且只实例化一次", t, func() {
		container := framework.NewHadeContainer()
		provider := &countingAgentProvider{agent: agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)}
		So(container.Bind(provider), ShouldBeNil)

		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)
		So(provider.instantiations(), ShouldEqual, 0)

		create := performRequest(engine, http.MethodPost, "/sessions", nil)
		So(create.Code, ShouldEqual, http.StatusCreated)
		So(provider.instantiations(), ShouldEqual, 1)

		for i := 0; i < 3; i++ {
			So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		}
		So(performRequest(engine, http.MethodGet, "/sessions/missing", nil).Code, ShouldEqual, http.StatusNotFound)
		So(provider.instantiations(), ShouldEqual, 1)

		// 示例工具只注册一次，重复请求不会让工具列表增长。
		So(provider.agent.ListTools(), ShouldHaveLength, 2)
		names := []string{provider.agent.ListTools()[0].Name, provider.agent.ListTools()[1].Name}
		So(names, ShouldResemble, []string{"echo", "time"})
	})
}

func TestNewAgentEngineSucceedsWithoutAgentBinding(t *testing.T) {
	Convey("容器里没有 Agent 时 Engine 仍能构造，请求返回 500 JSON 且可重试", t, func() {
		container := framework.NewHadeContainer()
		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)

		failed := performRequest(engine, http.MethodPost, "/sessions", nil)
		// 客户端只看到泛化文案，容器内部细节（含 key 名）不外泄。
		assertJSONError(failed, http.StatusInternalServerError, "agent service unavailable")
		So(failed.Body.String(), ShouldNotContainSubstring, contract.AgentKey)

		// 绑定后同一个 Engine 能恢复服务，说明失败没有被缓存。
		provider := &countingAgentProvider{agent: agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)}
		So(container.Bind(provider), ShouldBeNil)
		So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		So(provider.instantiations(), ShouldEqual, 1)
	})
}

func TestNewAgentEngineDoesNotTouchRealProviders(t *testing.T) {
	Convey("只绑定真实 Agent Provider（缺 LLM）：构造成功，首请求 500 JSON 且不 panic", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&agprovider.HadeAgentProvider{}), ShouldBeNil)

		// HadeAgentProvider.Params 会 MustMake(LLMKey)：构造期一旦实例化必然 panic。
		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)
		So(engine, ShouldNotBeNil)

		// Params 内的 MustMake panic 被 resolver 转成 error，而不是穿到
		// gin.Recovery 去生成一个空 body 的 500。
		var response *httptest.ResponseRecorder
		So(func() {
			response = performRequest(engine, http.MethodPost, "/sessions", nil)
		}, ShouldNotPanic)
		assertJSONError(response, http.StatusInternalServerError, "agent service unavailable")
		So(response.Body.String(), ShouldNotContainSubstring, "panic")

		// 补齐 LLM Provider 后同一个 Engine 恢复服务，说明失败没有被缓存。
		So(container.Bind(&llmp.HadeLLMProvider{}), ShouldBeNil)
		So(performRequest(engine, http.MethodGet, "/sessions/missing", nil).Code,
			ShouldEqual, http.StatusNotFound)
	})
}

func TestResolverRetriesToolRegistrationAfterPanic(t *testing.T) {
	Convey("示例工具注册 panic 不会让工具永久丢失，后续请求可重试", t, func() {
		agent := &flakyToolAgent{failRegister: true}
		engine := newTestEngine(t, agent)

		var first *httptest.ResponseRecorder
		So(func() {
			first = performRequest(engine, http.MethodPost, "/sessions", nil)
		}, ShouldNotPanic)
		assertJSONError(first, http.StatusInternalServerError, "agent service unavailable")
		So(agent.registeredNames(), ShouldBeEmpty)

		// 恢复正常后重试：工具补齐，请求成功。
		agent.setFailRegister(false)
		So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		So(agent.registeredNames(), ShouldResemble, []string{"echo", "time"})

		// 再来几次不会重复注册。
		for i := 0; i < 3; i++ {
			So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
		}
		So(agent.registeredNames(), ShouldResemble, []string{"echo", "time"})
	})
}

// flakyToolAgent 的 RegisterTool 可以按开关 panic，用来验证注册失败后的可重试性。
type flakyToolAgent struct {
	mu           sync.Mutex
	failRegister bool
	tools        []contract.ToolSpec
}

func (a *flakyToolAgent) setFailRegister(fail bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failRegister = fail
}

func (a *flakyToolAgent) registeredNames() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	names := make([]string, 0, len(a.tools))
	for _, spec := range a.tools {
		names = append(names, spec.Name)
	}
	return names
}

func (a *flakyToolAgent) RegisterTool(spec contract.ToolSpec, _ contract.ToolHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failRegister {
		panic("register tool exploded")
	}
	a.tools = append(a.tools, spec)
}

func (a *flakyToolAgent) ListTools() []contract.ToolSpec {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]contract.ToolSpec(nil), a.tools...)
}

func (*flakyToolAgent) CreateSession(context.Context) (string, error) { return "s1", nil }
func (*flakyToolAgent) GetSession(context.Context, string) (contract.Session, error) {
	return contract.Session{ID: "s1"}, nil
}
func (*flakyToolAgent) Run(context.Context, string, string, chan<- contract.AgentEvent) error {
	return nil
}
