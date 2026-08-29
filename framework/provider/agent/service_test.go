package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gohade/hade/framework/contract"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

// --- from service_session_test.go ---
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

// --- from service_tools_test.go ---
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

// --- from service_run_test.go ---
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

// toolCallResponse 构造一条只带工具调用的 LLM 响应。
func toolCallResponse(id, name, arguments string) contract.ChatResponse {
	return contract.ChatResponse{
		Message: contract.Message{
			Role:      "assistant",
			ToolCalls: []contract.ToolCall{{ID: id, Name: name, Arguments: arguments}},
		},
		Finish: contract.FinishToolCalls,
	}
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
				Message: contract.Message{
					Role:      "assistant",
					Content:   "call echo",
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
				},
				Finish: contract.FinishToolCalls,
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
			toolCallResponse("c1", "echo", `{}`),
			toolCallResponse("c2", "echo", `{}`),
			toolCallResponse("c3", "echo", `{}`),
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

func TestRun_SettlesDanglingToolCallsBeforeUserMessage(t *testing.T) {
	Convey("上一轮崩溃残留的 tool_calls 在本轮 Run 开始时补齐", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		held, err := a.store.TryBeginRun(context.Background(), id)
		So(err, ShouldBeNil)
		So(held.AppendWithin(a.limits.MaxHistoryBytes, settleReserveBytes([]contract.ToolCall{{ID: "c1"}}), contract.Message{
			Role:      "assistant",
			Content:   "thinking",
			ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}},
		}), ShouldBeNil)
		held.Release()

		events := make(chan contract.AgentEvent, 8)
		err = a.Run(context.Background(), id, "next", events)
		close(events)
		So(err, ShouldBeNil)
		session := mustSession(a, id)
		var toolMsgs int
		for _, message := range session.Messages {
			if message.Role == "tool" && message.ToolCallID == "c1" {
				toolMsgs++
				So(message.Content, ShouldEqual, settleReasonInternal)
			}
		}
		So(toolMsgs, ShouldEqual, 1)
	})
}

func TestRun_SessionBusy(t *testing.T) {
	Convey("busy", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		run, err := a.store.TryBeginRun(context.Background(), id)
		So(err, ShouldBeNil)
		ch := make(chan contract.AgentEvent, 2)
		err = a.Run(context.Background(), id, "two", ch)
		run.Release()
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
			toolCallResponse("c1", "echo", `{}`),
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

// --- from service_lock_test.go ---
func TestGetSession_NotBlockedByRunningRun(t *testing.T) {
	Convey("Run 长时间持有 runMu 时 GetSession 仍然快速返回", t, func() {
		llm := newCancelWaitLLM()
		a := NewMemoryAgent(llm, 8)
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "long run", events) }()

		select {
		case <-llm.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("LLM 未进入 Chat")
		}

		type snapshotResult struct {
			session contract.Session
			err     error
		}
		snapshot := make(chan snapshotResult, 1)
		go func() {
			session, err := a.GetSession(context.Background(), id)
			snapshot <- snapshotResult{session: session, err: err}
		}()
		select {
		case got := <-snapshot:
			So(got.err, ShouldBeNil)
			So(got.session.ID, ShouldEqual, id)
			// user 消息在 Run 开始时就已写入，读接口能看到。
			So(got.session.Messages, ShouldHaveLength, 1)
			So(got.session.Messages[0].Content, ShouldEqual, "long run")
		case <-time.After(time.Second):
			t.Fatal("GetSession 被正在运行的 Run 阻塞")
		}

		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)
	})
}

func TestMemoryAgent_ConcurrentReadWriteIsRaceFree(t *testing.T) {
	Convey("并发 Run / GetSession / RegisterTool / ListTools 无数据竞争", t, func() {
		script := &llmp.ScriptLLM{}
		for i := 0; i < 64; i++ {
			script.Responses = append(script.Responses, contract.ChatResponse{
				Message: contract.Message{Role: "assistant", Content: "final " + strconv.Itoa(i)},
				Finish:  contract.FinishStop,
			})
		}
		a := NewMemoryAgent(script, 8)

		ids := make([]string, 0, 8)
		for i := 0; i < 8; i++ {
			id, err := a.CreateSession(context.Background())
			So(err, ShouldBeNil)
			ids = append(ids, id)
		}

		// 断言统一放到 goroutine 之外，避免在并发上下文里调用 convey。
		var (
			mu      sync.Mutex
			failure []string
			wg      sync.WaitGroup
		)
		record := func(message string) {
			mu.Lock()
			failure = append(failure, message)
			mu.Unlock()
		}

		for _, id := range ids {
			sessionID := id
			wg.Add(3)
			go func() {
				defer wg.Done()
				events := make(chan contract.AgentEvent, 8)
				if err := a.Run(context.Background(), sessionID, "hi", events); err != nil &&
					err != contract.ErrSessionBusy {
					record("run: " + err.Error())
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					if _, err := a.GetSession(context.Background(), sessionID); err != nil {
						record("get: " + err.Error())
					}
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					a.RegisterTool(
						contract.ToolSpec{Name: "t" + strconv.Itoa(i%4)},
						func(context.Context, string) (string, error) { return "ok", nil },
					)
					_ = a.ListTools()
				}
			}()
		}
		wg.Wait()
		So(failure, ShouldBeEmpty)

		// 同名注册只覆盖不追加。
		So(a.ListTools(), ShouldHaveLength, 4)
		for _, id := range ids {
			assertHistoryValidForOpenAI(mustSession(a, id).Messages)
		}
	})
}

// --- from service_limits_test.go ---
func TestCreateSession_HitsSessionLimit(t *testing.T) {
	Convey("Session 数达到上限后返回 ErrSessionLimit", t, func() {
		a := NewMemoryAgentWithLimits(&fakeLLM{}, 8, Limits{MaxSessions: 2})
		first, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		second, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		So(first, ShouldNotEqual, second)

		_, err = a.CreateSession(context.Background())
		So(err, ShouldEqual, contract.ErrSessionLimit)

		// 已有 Session 仍然可用。
		_, err = a.GetSession(context.Background(), first)
		So(err, ShouldBeNil)
	})

	Convey("默认上限来自 contract 常量", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		So(a.limits.MaxSessions, ShouldEqual, contract.DefaultMaxSessions)
		So(a.limits.MaxMessageBytes, ShouldEqual, contract.DefaultMaxMessageBytes)
		So(a.limits.MaxHistoryBytes, ShouldEqual, contract.DefaultMaxHistoryBytes)
	})
}

func TestRun_RejectsOversizedUserMessage(t *testing.T) {
	Convey("单条 user 消息超限时不产生任何事件", t, func() {
		a := NewMemoryAgentWithLimits(&fakeLLM{}, 8, Limits{MaxMessageBytes: 16})
		id, _ := a.CreateSession(context.Background())
		events := make(chan contract.AgentEvent, 4)

		err := a.Run(context.Background(), id, strings.Repeat("x", 17), events)
		So(err, ShouldEqual, contract.ErrMessageTooLarge)
		So(len(events), ShouldEqual, 0)

		session := mustSession(a, id)
		So(session.Messages, ShouldHaveLength, 0)

		// 边界内的消息正常接受。
		So(a.Run(context.Background(), id, strings.Repeat("x", 16), events), ShouldBeNil)
	})
}

func TestRun_HistoryLimitRejectsOversizedLLMResponse(t *testing.T) {
	Convey("LLM 单次超大响应被拒绝，历史不越界", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{{
			Message: contract.Message{Role: "assistant", Content: strings.Repeat("y", 200)},
			Finish:  contract.FinishStop,
		}}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 16)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrHistoryLimit)

		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "history_limit")
		for _, event := range collected {
			So(event.Type, ShouldNotEqual, contract.EventFinal)
		}

		session := mustSession(a, id)
		So(session.Messages, ShouldHaveLength, 1)
		So(session.Messages[0].Role, ShouldEqual, "user")
		So(storeUsedBytes(a, id), ShouldBeLessThanOrEqualTo, 70)
	})
}

func TestRun_HistoryLimitRollsBackDanglingToolCalls(t *testing.T) {
	Convey("工具结果写不下时回滚整组 tool_calls，历史既不越界也不残留悬空 call", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "echo", `{}`),
		}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(context.Context, string) (string, error) {
			return strings.Repeat("z", 200), nil
		})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 16)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrHistoryLimit)
		collect(events)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(session.Messages, ShouldHaveLength, 1)
		So(storeUsedBytes(a, id), ShouldBeLessThanOrEqualTo, 70)
	})
}

func TestRun_CanceledSettlementStaysWithinHistoryLimit(t *testing.T) {
	Convey("补偿写入使用预留配额，取消后历史仍不越界", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "slow", `{}`),
		}}
		a := NewMemoryAgentWithLimits(script, 8, Limits{MaxHistoryBytes: 70})
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "unused", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "hi", events) }()
		So((<-events).Type, ShouldEqual, contract.EventSession)
		So((<-events).Type, ShouldEqual, contract.EventThought)
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{settleReasonCanceled})
		So(storeUsedBytes(a, id), ShouldBeLessThanOrEqualTo, 70)
	})
}

// --- from service_cancel_history_test.go ---
// cancelScenarioScript 第一轮返回一个 tool_call，第二轮直接收尾。
// 第二轮的 ChatRequest 就是"取消之后真正送给 LLM 的历史"。
func cancelScenarioScript() *llmp.ScriptLLM {
	return &llmp.ScriptLLM{Responses: []contract.ChatResponse{
		toolCallResponse("c1", "slow", `{"k":"v"}`),
		{Message: contract.Message{Role: "assistant", Content: "ok"}, Finish: contract.FinishStop},
	}}
}

func waitRunErr(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Run 取消后没有返回")
		return nil
	}
}

// assertSecondRunHistory 在同一个 Session 上再跑一轮，并断言送给 LLM 的历史合法。
func assertSecondRunHistory(t *testing.T, a *MemoryAgent, script *llmp.ScriptLLM, id string) {
	t.Helper()
	events := make(chan contract.AgentEvent, 32)
	done := make(chan error, 1)
	go func() {
		done <- a.Run(context.Background(), id, "again", events)
		close(events)
	}()
	So(waitRunErr(t, done), ShouldBeNil)
	collect(events)

	So(len(script.Calls), ShouldEqual, 2)
	assertHistoryValidForOpenAI(script.Calls[1].Messages)
	assertHistoryValidForOpenAI(mustSession(a, id).Messages)
}

func mustSession(a *MemoryAgent, id string) contract.Session {
	session, err := a.GetSession(context.Background(), id)
	So(err, ShouldBeNil)
	return session
}

func storeUsedBytes(a *MemoryAgent, id string) int {
	msgs, err := a.store.Open(context.Background(), id)
	So(err, ShouldBeNil)
	used := 0
	for _, message := range msgs {
		used += messageBytes(message)
	}
	return used
}

func TestRun_CancelBeforeActionSentClosesToolCall(t *testing.T) {
	Convey("取消发生在 action 事件发送前：pending 的 tool_call 被补 canceled", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "should not run", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		// 不带缓冲：读到 thought 后停止读取，Run 必然卡在 action 的 send 上。
		events := make(chan contract.AgentEvent)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "hi", events) }()

		So((<-events).Type, ShouldEqual, contract.EventSession)
		So((<-events).Type, ShouldEqual, contract.EventThought)
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{settleReasonCanceled})

		assertSecondRunHistory(t, a, script, id)
	})
}

func TestRun_CancelBeforeObservationSentKeepsRealResult(t *testing.T) {
	Convey("取消发生在 observation 事件发送前：保留真实结果且不重复补偿", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(context.Context, string) (string, error) {
			return "real result", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent)
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, id, "hi", events) }()

		So((<-events).Type, ShouldEqual, contract.EventSession)
		So((<-events).Type, ShouldEqual, contract.EventThought)
		So((<-events).Type, ShouldEqual, contract.EventAction)
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		// 真实结果先落历史再推事件，所以这里必须是工具结果而不是 canceled。
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{"real result"})

		assertSecondRunHistory(t, a, script, id)
	})
}

func TestRun_CancelDuringToolExecutionKeepsHistoryValid(t *testing.T) {
	Convey("取消发生在工具执行中途：历史依然闭环", t, func() {
		script := cancelScenarioScript()
		a := NewMemoryAgent(script, 8)
		entered := make(chan struct{})
		a.RegisterTool(contract.ToolSpec{Name: "slow"}, func(ctx context.Context, _ string) (string, error) {
			close(entered)
			<-ctx.Done()
			return "midway", nil
		})
		id, _ := a.CreateSession(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(ctx, id, "hi", events)
			close(events)
		}()

		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("工具未被调用")
		}
		cancel()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrCanceled)
		collect(events)

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		replies := toolReplies(session.Messages)["c1"]
		So(replies, ShouldHaveLength, 1)
		// 工具在取消后仍然返回了结果，这个结果或补偿值都必须闭环该 call。
		So(replies[0] == "midway" || replies[0] == settleReasonCanceled, ShouldBeTrue)

		assertSecondRunHistory(t, a, script, id)
	})
}

// --- from service_panic_test.go ---
// toolReplies 汇总历史里每个 tool_call_id 收到的回复内容。
func toolReplies(messages []contract.Message) map[string][]string {
	replies := map[string][]string{}
	for _, message := range messages {
		if message.Role == "tool" {
			replies[message.ToolCallID] = append(replies[message.ToolCallID], message.Content)
		}
	}
	return replies
}

// assertHistoryValidForOpenAI 断言历史满足 OpenAI 约束：
// 每个 assistant.tool_calls 里的 call id 都有且只有一条 tool 回复。
func assertHistoryValidForOpenAI(messages []contract.Message) {
	replies := toolReplies(messages)
	calls := 0
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		for _, call := range message.ToolCalls {
			calls++
			So(replies[call.ID], ShouldHaveLength, 1)
		}
	}
	// 不允许出现没有对应 assistant.tool_call 的孤立 tool 消息。
	So(len(replies), ShouldEqual, calls)
}

func TestRun_ToolPanicBecomesObservationAndLoopReachesFinal(t *testing.T) {
	Convey("第三方工具 panic 转成 error observation，循环仍然走到 final", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			toolCallResponse("c1", "boom", `{}`),
			{Message: contract.Message{Role: "assistant", Content: "recovered"}, Finish: contract.FinishStop},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "boom"}, func(context.Context, string) (string, error) {
			panic("tool exploded")
		})
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 卡死")
		}
		So(runErr, ShouldBeNil)

		collected := collect(events)
		types := make([]string, len(collected))
		var observation string
		for i, event := range collected {
			types[i] = event.Type
			if event.Type == contract.EventObservation {
				observation, _ = event.Data["content"].(string)
			}
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		So(observation, ShouldContainSubstring, "panicked")
		So(observation, ShouldContainSubstring, "tool exploded")

		session, _ := a.GetSession(context.Background(), id)
		assertHistoryValidForOpenAI(session.Messages)
	})
}

func TestRun_PanicInsideRunSettlesPendingToolCalls(t *testing.T) {
	Convey("Run 自身 panic 时补齐未闭环 tool_call 并返回 ErrInternal", t, func() {
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{{
			Message: contract.Message{
				Role:    "assistant",
				Content: "two calls",
				ToolCalls: []contract.ToolCall{
					{ID: "c1", Name: "closer", Arguments: `{}`},
					{ID: "c2", Name: "closer", Arguments: `{}`},
				},
			},
			Finish: contract.FinishToolCalls,
		}}}
		a := NewMemoryAgent(script, 8)
		id, _ := a.CreateSession(context.Background())

		events := make(chan contract.AgentEvent, 32)
		// 模拟调用方提前关闭事件 channel：后续 send 会 panic。
		a.RegisterTool(contract.ToolSpec{Name: "closer"}, func(context.Context, string) (string, error) {
			close(events)
			return "ok", nil
		})

		done := make(chan error, 1)
		go func() { done <- a.Run(context.Background(), id, "go", events) }()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 未从 panic 中恢复")
		}
		So(runErr, ShouldEqual, contract.ErrInternal)

		session, _ := a.GetSession(context.Background(), id)
		assertHistoryValidForOpenAI(session.Messages)
		replies := toolReplies(session.Messages)
		So(replies["c1"][0], ShouldEqual, "ok")
		So(replies["c2"][0], ShouldEqual, settleReasonInternal)
	})
}

func TestRun_PanicInLLMReturnsInternal(t *testing.T) {
	Convey("LLM 实现 panic 时 Run 返回 ErrInternal 且发 internal 事件", t, func() {
		a := NewMemoryAgent(&panicLLM{}, 8)
		id, _ := a.CreateSession(context.Background())
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()

		var runErr error
		select {
		case runErr = <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Run 未从 panic 中恢复")
		}
		So(runErr, ShouldEqual, contract.ErrInternal)

		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "internal")
	})
}

type panicLLM struct{}

func (*panicLLM) Chat(context.Context, contract.ChatRequest) (contract.ChatResponse, error) {
	panic("llm exploded")
}

// --- from service_diagnostics_test.go ---
func TestRun_PanicIsLoggedWithValueAndStack(t *testing.T) {
	Convey("Run recover 把 panic value 与调用栈写进 contract.Log，但不外泄给客户端", t, func() {
		log := &captureLog{}
		a := NewMemoryAgent(&panicLLM{}, 8)
		a.logger = log

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 8)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "go", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldEqual, contract.ErrInternal)

		So(len(log.errors), ShouldEqual, 1)
		entry := log.errors[0]
		So(entry.msg, ShouldEqual, "agent run panic")
		So(entry.fields["session"], ShouldEqual, id)
		So(fmt.Sprint(entry.fields["value"]), ShouldContainSubstring, "llm exploded")
		stack := fmt.Sprint(entry.fields["stack"])
		So(stack, ShouldContainSubstring, "goroutine")
		So(stack, ShouldContainSubstring, "framework/provider/agent")
		So(stack, ShouldContainSubstring, "MemoryAgent).Run")

		collected := collect(events)
		last := collected[len(collected)-1]
		So(last.Type, ShouldEqual, contract.EventError)
		So(last.Data["code"], ShouldEqual, "internal")
		So(fmt.Sprint(last.Data), ShouldNotContainSubstring, "llm exploded")
		So(fmt.Sprint(last.Data), ShouldNotContainSubstring, "goroutine")
	})

	Convey("未注入 Log 时 logRunPanic 静默跳过且不 panic", t, func() {
		a := NewMemoryAgent(&fakeLLM{}, 8)
		So(a.logger, ShouldBeNil)
		So(func() { a.logRunPanic(context.Background(), "sid", "boom") }, ShouldNotPanic)
	})
}

type captureLog struct {
	errors []captureLogEntry
}

type captureLogEntry struct {
	msg    string
	fields map[string]interface{}
}

func (c *captureLog) Panic(context.Context, string, map[string]interface{}) {}
func (c *captureLog) Fatal(context.Context, string, map[string]interface{}) {}
func (c *captureLog) Error(_ context.Context, msg string, fields map[string]interface{}) {
	c.errors = append(c.errors, captureLogEntry{msg: msg, fields: fields})
}
func (c *captureLog) Warn(context.Context, string, map[string]interface{})  {}
func (c *captureLog) Info(context.Context, string, map[string]interface{})  {}
func (c *captureLog) Debug(context.Context, string, map[string]interface{}) {}
func (c *captureLog) Trace(context.Context, string, map[string]interface{}) {}
func (c *captureLog) SetLevel(contract.LogLevel)                            {}
func (c *captureLog) SetCtxFielder(contract.CtxFielder)                     {}
func (c *captureLog) SetFormatter(contract.Formatter)                       {}
func (c *captureLog) SetOutput(io.Writer)                                   {}

func TestGetSession_TruncatesToolCallArguments(t *testing.T) {
	Convey("GetSession 同时截断 Content 与 tool arguments", t, func() {
		hugeArguments := `{"text":"` + strings.Repeat("z", contract.ContentMaxBytes*2) + `"}`
		script := &llmp.ScriptLLM{Responses: []contract.ChatResponse{
			{
				Message: contract.Message{
					Role:      "assistant",
					Content:   strings.Repeat("c", contract.ContentMaxBytes*2),
					ToolCalls: []contract.ToolCall{{ID: "c1", Name: "echo", Arguments: hugeArguments}},
				},
				Finish: contract.FinishToolCalls,
			},
			{Message: contract.Message{Role: "assistant", Content: "done"}, Finish: contract.FinishStop},
		}}
		a := NewMemoryAgent(script, 8)
		a.RegisterTool(contract.ToolSpec{Name: "echo"}, func(context.Context, string) (string, error) {
			return "ok", nil
		})

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hi", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldBeNil)
		collect(events)

		session := mustSession(a, id)
		var assistantCalls []contract.ToolCall
		for _, message := range session.Messages {
			So(len(message.Content), ShouldBeLessThanOrEqualTo, contract.ContentMaxBytes)
			if message.Role == "assistant" && len(message.ToolCalls) > 0 {
				assistantCalls = message.ToolCalls
			}
		}
		So(assistantCalls, ShouldHaveLength, 1)
		So(len(assistantCalls[0].Arguments), ShouldEqual, contract.ContentMaxBytes)
		So(len(hugeArguments), ShouldBeGreaterThan, contract.ContentMaxBytes)

		// 快照被截断，内部历史仍是完整值（下一轮要原样送回 LLM）。
		msgs, err := a.store.Open(context.Background(), id)
		So(err, ShouldBeNil)
		So(len(msgs[1].ToolCalls[0].Arguments), ShouldEqual, len(hugeArguments))
	})
}

// --- from service_openai_loop_test.go ---
const (
	openAIToolCallBody = `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant",` +
		`"content":null,"tool_calls":[{"id":"c1","type":"function",` +
		`"function":{"name":"echo","arguments":"{\"text\":\"hi\"}"}}]}}]}`
	openAIStopBody = `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}]}`
)

func TestRun_ClosedLoopAgainstOpenAIServer(t *testing.T) {
	Convey("对接返回 200 的 OpenAI 兼容服务，ReAct 闭环完成且第二轮请求携带合法历史", t, func() {
		var (
			mu     sync.Mutex
			bodies [][]byte
		)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			mu.Lock()
			bodies = append(bodies, body)
			round := len(bodies)
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if round == 1 {
				_, _ = io.WriteString(w, openAIToolCallBody)
				return
			}
			_, _ = io.WriteString(w, openAIStopBody)
		}))
		defer server.Close()

		a := NewMemoryAgent(llmp.NewOpenAI(server.URL, "secret-key", "fixture-model"), 8)
		a.RegisterTool(contract.ToolSpec{
			Name: "echo",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"text": map[string]interface{}{"type": "string"}},
			},
		}, func(_ context.Context, argsJSON string) (string, error) {
			return argsJSON, nil
		})

		id, err := a.CreateSession(context.Background())
		So(err, ShouldBeNil)
		events := make(chan contract.AgentEvent, 32)
		done := make(chan error, 1)
		go func() {
			done <- a.Run(context.Background(), id, "hello", events)
			close(events)
		}()
		So(waitRunErr(t, done), ShouldBeNil)

		collected := collect(events)
		types := make([]string, len(collected))
		for i, event := range collected {
			types[i] = event.Type
		}
		So(types, ShouldResemble, []string{
			contract.EventSession, contract.EventThought, contract.EventAction,
			contract.EventObservation, contract.EventThought, contract.EventFinal,
		})
		So(collected[len(collected)-1].Data["content"], ShouldEqual, "done")

		session := mustSession(a, id)
		assertHistoryValidForOpenAI(session.Messages)
		So(toolReplies(session.Messages)["c1"], ShouldResemble, []string{`{"text":"hi"}`})

		mu.Lock()
		defer mu.Unlock()
		So(bodies, ShouldHaveLength, 2)
		second := string(bodies[1])
		So(second, ShouldContainSubstring, `"model":"fixture-model"`)
		So(second, ShouldContainSubstring, `"tool_calls":[{"id":"c1","type":"function"`)
		So(second, ShouldContainSubstring, `"role":"tool"`)
		So(second, ShouldContainSubstring, `"tool_call_id":"c1"`)
	})
}
