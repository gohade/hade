package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	configprovider "github.com/gohade/hade/framework/provider/config"
	llmp "github.com/gohade/hade/framework/provider/llm"
	logprovider "github.com/gohade/hade/framework/provider/log"
	. "github.com/smartystreets/goconvey/convey"
)

func bindLLMFor(container framework.Container) {
	So(container.Bind(&llmp.HadeLLMProvider{}), ShouldBeNil)
}

func TestHadeAgentProvider_ParamsDefaults(t *testing.T) {
	Convey("Provider 元数据与默认参数", t, func() {
		provider := &HadeAgentProvider{}
		So(provider.Name(), ShouldEqual, contract.AgentKey)
		So(provider.IsDefer(), ShouldBeTrue)

		container := framework.NewHadeContainer()
		bindLLMFor(container)
		params := provider.Params(container)
		So(params, ShouldHaveLength, 4)
		So(params[1], ShouldEqual, contract.DefaultMaxIter)
		So(params[3], ShouldBeNil)
	})

	Convey("没有配置时仍能构造 Agent", t, func() {
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		instance, err := NewHadeAgentService((&HadeAgentProvider{}).Params(container)...)
		So(err, ShouldBeNil)
		agent, ok := instance.(*AgentRuntime)
		So(ok, ShouldBeTrue)
		So(agent.maxIter, ShouldEqual, contract.DefaultMaxIter)
	})

	Convey("第三个参数注入 contract.Log", t, func() {
		log := &captureLog{}
		instance, err := NewHadeAgentService(&fakeLLM{}, 4, log)
		So(err, ShouldBeNil)
		So(instance.(*AgentRuntime).logger, ShouldEqual, log)
	})

	Convey("旧的两参数调用仍然可用", t, func() {
		instance, err := NewHadeAgentService(&fakeLLM{}, 4)
		So(err, ShouldBeNil)
		agent := instance.(*AgentRuntime)
		So(agent.maxIter, ShouldEqual, 4)
		So(agent.logger, ShouldBeNil)
	})

	Convey("容器绑定 Log 后 Params 会传入 logger", t, func() {
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		So(container.Bind(&logprovider.HadeTestingLogProvider{}), ShouldBeNil)
		params := (&HadeAgentProvider{}).Params(container)
		So(params[2], ShouldNotBeNil)
		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		So(instance.(*AgentRuntime).logger, ShouldNotBeNil)
	})
}

func TestHadeAgentProvider_ParamsReadsMaxIterationsFromConfig(t *testing.T) {
	Convey("agent.yaml 中的 max_iterations 被读入", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content: []byte(`
max_iterations: 3
`),
		}), ShouldBeNil)
		bindLLMFor(container)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[1], ShouldEqual, 3)

		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		agent := instance.(*AgentRuntime)
		So(agent.maxIter, ShouldEqual, 3)
	})
}

type stubRedisService struct {
	client *redis.Client
	err    error
}

func (s *stubRedisService) GetClient(...contract.RedisOption) (*redis.Client, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

type stubRedisProvider struct {
	svc contract.RedisService
}

func (p *stubRedisProvider) Name() string { return contract.RedisKey }
func (p *stubRedisProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return p.svc, nil }
}
func (p *stubRedisProvider) Boot(framework.Container) error           { return nil }
func (p *stubRedisProvider) IsDefer() bool                            { return false }
func (p *stubRedisProvider) Params(framework.Container) []interface{} { return nil }

func TestHadeAgentProvider_InjectsRedisStore(t *testing.T) {
	Convey("仅绑定 Redis、未配置 session_store 时默认内存，不注入 Redis store", t, func() {
		_, client := miniClient(t)
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		So(container.Bind(&stubRedisProvider{svc: &stubRedisService{client: client}}), ShouldBeNil)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[3], ShouldBeNil)
	})

	Convey("session_store=redis 且 Ping 成功时注入 Redis store，Create 写进 miniredis", t, func() {
		_, client := miniClient(t)
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content:  []byte("session_store: redis\n"),
		}), ShouldBeNil)
		bindLLMFor(container)
		So(container.Bind(&stubRedisProvider{svc: &stubRedisService{client: client}}), ShouldBeNil)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[3], ShouldNotBeNil)
		_, isStore := params[3].(SessionStore)
		So(isStore, ShouldBeTrue)

		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		agent := instance.(*AgentRuntime)
		id, err := agent.CreateSession(context.Background())
		So(err, ShouldBeNil)
		other := newRedisStore(client)
		msgs, err := other.Open(context.Background(), id)
		So(err, ShouldBeNil)
		So(msgs, ShouldBeEmpty)
	})

	Convey("session_store=redis 但 GetClient 失败时回退内存，构造仍成功", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content:  []byte("session_store: redis\n"),
		}), ShouldBeNil)
		bindLLMFor(container)
		So(container.Bind(&stubRedisProvider{svc: &stubRedisService{err: errors.New("dial")}}), ShouldBeNil)
		params := (&HadeAgentProvider{}).Params(container)
		So(params[3], ShouldBeNil)
		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		_, isMemory := instance.(*AgentRuntime).store.(*memoryStore)
		So(isMemory, ShouldBeTrue)
	})
}
