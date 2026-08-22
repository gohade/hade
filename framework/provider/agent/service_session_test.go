package agent

import (
	"context"
	"testing"

	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryAgent_CreateAndGetSession(t *testing.T) {
	Convey("session crud", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		So(id, ShouldNotBeEmpty)
		s, err := a.GetSession(context.Background(), id)
		So(err, ShouldBeNil)
		So(s.ID, ShouldEqual, id)
		So(len(s.Messages), ShouldEqual, 0)
		_, err = a.GetSession(context.Background(), "missing")
		So(err, ShouldEqual, contract.ErrSessionNotFound)
	})
}

func TestMemoryAgent_RegisterEchoTool(t *testing.T) {
	Convey("tools", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		a.RegisterTool(contract.ToolSpec{
			Name:        "echo",
			Description: "echo text",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{"type": "string"},
				},
			},
		}, func(ctx context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})
		So(len(a.ListTools()), ShouldEqual, 1)
		So(a.ListTools()[0].Name, ShouldEqual, "echo")
	})
}

type fakeLLM struct{}

func (f *fakeLLM) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	return contract.ChatResponse{Finish: contract.FinishStop}, nil
}
