package agent

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/gohade/hade/framework"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	configprovider "github.com/gohade/hade/framework/provider/config"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAgentEngineConsoleConfig(t *testing.T) {
	Convey("Console 默认开启", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&agentStub{agent: agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)}), ShouldBeNil)
		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)
		So(performRequest(engine, http.MethodGet, "/", nil).Code, ShouldEqual, http.StatusOK)
	})

	Convey("显式关闭 Console 时只保留业务 Session API", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&configprovider.FakeConfigProvider{
			FileName: "agent", Content: []byte("console_enabled: false\n"),
		}), ShouldBeNil)
		So(container.Bind(&agentStub{agent: agprovider.NewAgentRuntime(&llmp.ScriptLLM{}, 8)}), ShouldBeNil)

		engine, err := NewAgentEngine(container)
		So(err, ShouldBeNil)
		for _, path := range []string{"/", "/agent-ui.css", "/agent-ui.js", "/config", "/tools"} {
			So(performRequest(engine, http.MethodGet, path, nil).Code, ShouldEqual, http.StatusNotFound)
		}
		So(performRequest(engine, http.MethodPut, "/config", bytes.NewBufferString(`{}`)).Code,
			ShouldEqual, http.StatusNotFound)
		So(performRequest(engine, http.MethodPost, "/sessions", nil).Code, ShouldEqual, http.StatusCreated)
	})
}
