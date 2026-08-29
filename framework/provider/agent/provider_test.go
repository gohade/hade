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
		So(params, ShouldHaveLength, 5)
		So(params[1], ShouldEqual, contract.DefaultMaxIter)
		So(params[2], ShouldResemble, DefaultLimits())
		So(params[4], ShouldBeNil)
	})

	Convey("没有配置时构造出的 Agent 使用默认上限", t, func() {
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		instance, err := NewHadeAgentService((&HadeAgentProvider{}).Params(container)...)
		So(err, ShouldBeNil)
		agent, ok := instance.(*MemoryAgent)
		So(ok, ShouldBeTrue)
		So(agent.limits, ShouldResemble, DefaultLimits())
	})

	Convey("第四个参数注入 contract.Log", t, func() {
		log := &captureLog{}
		instance, err := NewHadeAgentService(&fakeLLM{}, 4, DefaultLimits(), log)
		So(err, ShouldBeNil)
		So(instance.(*MemoryAgent).logger, ShouldEqual, log)
	})

	Convey("旧的两参数调用仍然可用，回退默认上限", t, func() {
		instance, err := NewHadeAgentService(&fakeLLM{}, 4)
		So(err, ShouldBeNil)
		agent := instance.(*MemoryAgent)
		So(agent.maxIter, ShouldEqual, 4)
		So(agent.limits, ShouldResemble, DefaultLimits())
		So(agent.logger, ShouldBeNil)
	})

	Convey("容器绑定 Log 后 Params 会传入 logger", t, func() {
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		So(container.Bind(&logprovider.HadeTestingLogProvider{}), ShouldBeNil)
		params := (&HadeAgentProvider{}).Params(container)
		So(params[3], ShouldNotBeNil)
		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		So(instance.(*MemoryAgent).logger, ShouldNotBeNil)
	})
}

func TestHadeAgentProvider_ParamsReadsLimitsFromConfig(t *testing.T) {
	Convey("agent.yaml 中的上限被读入 Limits", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content: []byte(`
max_iterations: 3
max_sessions: 12
max_message_bytes: 2048
max_history_bytes: 4096
`),
		}), ShouldBeNil)
		bindLLMFor(container)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[1], ShouldEqual, 3)
		So(params[2], ShouldResemble, Limits{
			MaxSessions:     12,
			MaxMessageBytes: 2048,
			MaxHistoryBytes: 4096,
		})

		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		agent := instance.(*MemoryAgent)
		So(agent.maxIter, ShouldEqual, 3)
		So(agent.limits.MaxSessions, ShouldEqual, 12)
		So(agent.limits.MaxMessageBytes, ShouldEqual, 2048)
		So(agent.limits.MaxHistoryBytes, ShouldEqual, 4096)
	})

	Convey("仓库内三套环境配置的键与默认值一致", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content: []byte(`
port: ":8889"
max_iterations: 8
max_sessions: 1000
max_message_bytes: 65536
max_history_bytes: 1048576
`),
		}), ShouldBeNil)
		bindLLMFor(container)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[1], ShouldEqual, contract.DefaultMaxIter)
		// 配置值与代码默认值必须一致：改配置不应该悄悄改变现有行为。
		So(params[2], ShouldResemble, DefaultLimits())
	})

	Convey("配置里的非正值回退默认，不会构造出无界 Agent", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent",
			Content: []byte(`
max_sessions: 0
max_message_bytes: -1
max_history_bytes: 0
`),
		}), ShouldBeNil)
		bindLLMFor(container)

		So((&HadeAgentProvider{}).Params(container)[2], ShouldResemble, DefaultLimits())
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
	Convey("绑定 Redis 后 Params 注入 Redis store，Create 写进 miniredis", t, func() {
		_, client := miniClient(t)
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		So(container.Bind(&stubRedisProvider{svc: &stubRedisService{client: client}}), ShouldBeNil)

		params := (&HadeAgentProvider{}).Params(container)
		So(params[4], ShouldNotBeNil)
		_, isStore := params[4].(SessionStore)
		So(isStore, ShouldBeTrue)

		instance, err := NewHadeAgentService(params...)
		So(err, ShouldBeNil)
		agent := instance.(*MemoryAgent)
		id, err := agent.CreateSession(context.Background())
		So(err, ShouldBeNil)
		other := newRedisStore(client, DefaultLimits().MaxSessions)
		msgs, err := other.Open(context.Background(), id)
		So(err, ShouldBeNil)
		So(msgs, ShouldBeEmpty)
	})

	Convey("GetClient 失败时 NewHadeAgentService 返回 error，不退回内存", t, func() {
		container := framework.NewHadeContainer()
		bindLLMFor(container)
		So(container.Bind(&stubRedisProvider{svc: &stubRedisService{err: errors.New("dial")}}), ShouldBeNil)
		params := (&HadeAgentProvider{}).Params(container)
		_, err := NewHadeAgentService(params...)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "dial")
	})
}
