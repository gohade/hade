package agent

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRegisterTool_ValidatesAndDeduplicates(t *testing.T) {
	Convey("空名称与 nil handler 的注册被忽略", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		a.RegisterTool(contract.ToolSpec{Name: "  "}, func(context.Context, string) (string, error) {
			return "", nil
		})
		a.RegisterTool(contract.ToolSpec{Name: "nil-handler"}, nil)
		So(a.ListTools(), ShouldBeEmpty)
	})

	Convey("名称被去空格，同名注册覆盖而不追加", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		a.RegisterTool(contract.ToolSpec{Name: " echo ", Description: "first"},
			func(context.Context, string) (string, error) { return "first", nil })
		a.RegisterTool(contract.ToolSpec{Name: "echo", Description: "second"},
			func(context.Context, string) (string, error) { return "second", nil })

		tools := a.ListTools()
		So(tools, ShouldHaveLength, 1)
		So(tools[0].Name, ShouldEqual, "echo")
		So(tools[0].Description, ShouldEqual, "second")

		observation, err := a.execTool(context.Background(), "echo", `{}`)
		So(err, ShouldBeNil)
		So(observation, ShouldEqual, "second")
	})
}

func TestRegisterTool_DeepCopiesParameters(t *testing.T) {
	Convey("Parameters 被深拷贝，注册方后续修改不影响 Agent", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		properties := map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		}
		parameters := map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   []interface{}{"text"},
		}
		a.RegisterTool(contract.ToolSpec{Name: "echo", Parameters: parameters},
			func(context.Context, string) (string, error) { return "ok", nil })

		// 注册方改自己那份，Agent 内部不应受影响。
		parameters["type"] = "mutated"
		properties["text"] = "mutated"
		parameters["required"] = []interface{}{"mutated"}

		stored := a.ListTools()[0].Parameters
		So(stored["type"], ShouldEqual, "object")
		So(stored["properties"], ShouldResemble, map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		})
		So(stored["required"], ShouldResemble, []interface{}{"text"})

		// ListTools 返回的也是副本。
		stored["type"] = "caller-mutated"
		So(a.ListTools()[0].Parameters["type"], ShouldEqual, "object")
	})
}

func TestTruncate_IsUTF8Safe(t *testing.T) {
	Convey("按字节裁剪不切断多字节字符", t, func() {
		So(truncate("abc", 10), ShouldEqual, "abc")
		So(truncate("abc", 0), ShouldEqual, "")
		So(truncate("abc", -1), ShouldEqual, "")
		// "你" 占 3 字节：限 4 字节时只能保留第一个字，不能产生半个字符。
		So(truncate("你好", 4), ShouldEqual, "你")
		So(truncate("你好", 3), ShouldEqual, "你")
		So(truncate("你好", 2), ShouldEqual, "")
		So(truncate("你好", 6), ShouldEqual, "你好")
	})
}

func TestExecTool_UnknownToolReturnsError(t *testing.T) {
	Convey("未注册的工具返回错误而不是 panic", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		_, err := a.execTool(context.Background(), "missing", `{}`)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "unknown tool: missing")
	})
}
