package agent

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

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
