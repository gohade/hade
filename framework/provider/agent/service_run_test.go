package agent

import (
	"context"
	"testing"
	"time"

	llmp "github.com/gohade/hade/framework/provider/llm"
	"github.com/gohade/hade/framework/contract"
	. "github.com/smartystreets/goconvey/convey"
)

type cancelWaitLLM struct {
	entered chan struct{}
}

func newCancelWaitLLM() *cancelWaitLLM {
	return &cancelWaitLLM{entered: make(chan struct{})}
}

func (c *cancelWaitLLM) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	close(c.entered)
	<-ctx.Done()
	return contract.ChatResponse{}, ctx.Err()
}

func collect(ch <-chan contract.AgentEvent) []contract.AgentEvent {
	var out []contract.AgentEvent
	for e := range ch {
		out = append(out, e)
	}
	return out
}

func TestRun_EchoReActThenFinal(t *testing.T) {
	Convey("react echo", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message:   contract.Message{Role: "assistant", Content: "call echo"},
				ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				Finish:    contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "ok"},
				Finish:  contract.FinishStop,
			},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 16)
		go func() {
			_ = a.Run(context.Background(), id, "hello", ch)
			close(ch)
		}()
		evs := collect(ch)
		types := make([]string, len(evs))
		for i, e := range evs {
			types[i] = e.Type
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		sess, _ := a.GetSession(context.Background(), id)
		So(len(sess.Messages) >= 2, ShouldBeTrue)
		So(sess.Messages[0].Role, ShouldEqual, "user")
		So(sess.Messages[0].Content, ShouldEqual, "hello")
	})
}

func TestRun_MaxIterations(t *testing.T) {
	Convey("max iter", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
			{ToolCalls: []contract.ToolCall{{ID: "c2", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
			{ToolCalls: []contract.ToolCall{{ID: "c3", Name: "echo", Arguments: `{}`}}, Finish: contract.FinishToolCalls},
		}}
		a := NewMemoryAgent(script, 2)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			return "x", nil
		})
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 32)
		var runErr error
		go func() {
			runErr = a.Run(context.Background(), id, "go", ch)
			close(ch)
		}()
		evs := collect(ch)
		So(runErr, ShouldEqual, contract.ErrMaxIterations)
		last := evs[len(evs)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "max_iterations")
		for _, e := range evs {
			So(e.Type, ShouldNotEqual, contract.EventFinal)
		}
	})
}

func TestRun_SessionBusy(t *testing.T) {
	Convey("busy", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		a.mu.RLock()
		s := a.sess[id]
		a.mu.RUnlock()
		s.mu.Lock()
		ch := make(chan contract.AgentEvent, 2)
		err := a.Run(context.Background(), id, "two", ch)
		s.mu.Unlock()
		So(err, ShouldEqual, contract.ErrSessionBusy)
	})
}

func TestRun_CanceledWhenLLMReturnsAfterCtxDone(t *testing.T) {
	Convey("canceled not llm_failed", t, func() {
		llm := newCancelWaitLLM()
		a := NewMemoryAgent(llm, 8)
		id, _ := a.CreateSession(context.Background())
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan contract.AgentEvent, 8)
		done := make(chan struct{})
		var runErr error
		go func() {
			runErr = a.Run(ctx, id, "cancel me", ch)
			close(ch)
			close(done)
		}()
		select {
		case <-llm.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("LLM Chat never entered")
		}
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run did not finish after cancel")
		}
		evs := collect(ch)
		So(runErr, ShouldEqual, contract.ErrCanceled)
		for _, e := range evs {
			if e.Type == contract.EventError {
				So(e.Data["code"], ShouldNotEqual, "llm_failed")
			}
		}
		last := evs[len(evs)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "canceled")
	})
}

func TestRun_ToolHandlerCanRegisterToolWithoutDeadlock(t *testing.T) {
	Convey("tool reentrant register", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}},
				Finish:    contract.FinishToolCalls,
			},
			{
				Message: contract.Message{Role: "assistant", Content: "done"},
				Finish:  contract.FinishStop,
			},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(ctx context.Context, argsJSON string) (string, error) {
			a.RegisterTool(contract.ToolSpec{Name: "nested"}, func(ctx context.Context, argsJSON string) (string, error) {
				return "nested", nil
			})
			return "ok", nil
		})
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 16)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", ch)
			close(ch)
		}()
		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run deadlocked when tool handler called RegisterTool")
		}
		So(runErr, ShouldBeNil)
		evs := collect(ch)
		hasFinal := false
		for _, e := range evs {
			if e.Type == contract.EventFinal {
				hasFinal = true
			}
		}
		So(hasFinal, ShouldBeTrue)
	})
}

func TestRun_EmptyMessageNoEvents(t *testing.T) {
	Convey("empty message", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 4)
		err := a.Run(context.Background(), id, "  ", ch)
		So(err, ShouldEqual, contract.ErrEmptyMessage)
		So(len(ch), ShouldEqual, 0)
	})
}

func TestRun_SessionNotFoundNoEvents(t *testing.T) {
	Convey("session not found", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		ch := make(chan contract.AgentEvent, 4)
		err := a.Run(context.Background(), "missing", "hi", ch)
		So(err, ShouldEqual, contract.ErrSessionNotFound)
		So(len(ch), ShouldEqual, 0)
	})
}

func TestRun_KeepUserOnLLMError(t *testing.T) {
	Convey("keep user", t, func() {
		script := &llmp.ScriptLLM{Errs: []error{contract.ErrLLMFailed}}
		a := NewMemoryAgent(script, 8)
		id, _ := a.CreateSession(context.Background())
		ch := make(chan contract.AgentEvent, 8)
		go func() {
			_ = a.Run(context.Background(), id, "keepme", ch)
			close(ch)
		}()
		collect(ch)
		sess, _ := a.GetSession(context.Background(), id)
		So(sess.Messages[0].Content, ShouldEqual, "keepme")
	})
}
