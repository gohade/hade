package agent

import (
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func toolNames(agent contract.Agent) map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range agent.ListTools() {
		out[spec.Name] = struct{}{}
	}
	return out
}

func TestRegisterExampleTools_SkipsUserToolsWithoutORM(t *testing.T) {
	Convey("未绑定 ORM 时只有 echo 和 time", t, func() {
		container := framework.NewHadeContainer()
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "echo")
		So(names, ShouldContainKey, "time")
		So(names, ShouldNotContainKey, "create_user")
		So(names, ShouldNotContainKey, "get_user")
		So(names, ShouldNotContainKey, "list_users")
	})
}

type stubORMProvider struct{}

func (p *stubORMProvider) Name() string { return contract.ORMKey }
func (p *stubORMProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return struct{}{}, nil }
}
func (p *stubORMProvider) Boot(framework.Container) error           { return nil }
func (p *stubORMProvider) IsDefer() bool                            { return true }
func (p *stubORMProvider) Params(framework.Container) []interface{} { return nil }

func TestRegisterExampleTools_RegistersUserToolsWhenORMBound(t *testing.T) {
	Convey("绑定 ORM 关键字后注册三个 User 工具", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&stubORMProvider{}), ShouldBeNil)
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "create_user")
		So(names, ShouldContainKey, "get_user")
		So(names, ShouldContainKey, "list_users")
	})
}
