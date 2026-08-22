package agent

import (
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	configprovider "github.com/gohade/hade/framework/provider/config"
	llmp "github.com/gohade/hade/framework/provider/llm"
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
		So(params, ShouldHaveLength, 3)
		So(params[1], ShouldEqual, contract.DefaultMaxIter)
		So(params[2], ShouldResemble, DefaultLimits())
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

	Convey("旧的两参数调用仍然可用，回退默认上限", t, func() {
		instance, err := NewHadeAgentService(&fakeLLM{}, 4)
		So(err, ShouldBeNil)
		agent := instance.(*MemoryAgent)
		So(agent.maxIter, ShouldEqual, 4)
		So(agent.limits, ShouldResemble, DefaultLimits())
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
