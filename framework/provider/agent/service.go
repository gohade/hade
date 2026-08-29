package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
)

var _ contract.Agent = (*MemoryAgent)(nil)

// Limits 是 MemoryAgent 的有界资源配置，全部为硬上限，不依赖后台清理协程。
type Limits struct {
	// MaxSessions 是并存 Session 数量上限，超出后 CreateSession 返回 ErrSessionLimit。
	MaxSessions int
	// MaxMessageBytes 是单条 user 消息字节上限，超出后 Run 返回 ErrMessageTooLarge。
	MaxMessageBytes int
	// MaxHistoryBytes 是单个 Session 历史字节上限，超出后写入被拒绝并返回 ErrHistoryLimit。
	MaxHistoryBytes int
}

// DefaultLimits 返回生产环境默认上限。
func DefaultLimits() Limits {
	return Limits{
		MaxSessions:     contract.DefaultMaxSessions,
		MaxMessageBytes: contract.DefaultMaxMessageBytes,
		MaxHistoryBytes: contract.DefaultMaxHistoryBytes,
	}
}

func (l Limits) normalize() Limits {
	defaults := DefaultLimits()
	if l.MaxSessions <= 0 {
		l.MaxSessions = defaults.MaxSessions
	}
	if l.MaxMessageBytes <= 0 {
		l.MaxMessageBytes = defaults.MaxMessageBytes
	}
	if l.MaxHistoryBytes <= 0 {
		l.MaxHistoryBytes = defaults.MaxHistoryBytes
	}
	return l
}

type registeredTool struct {
	spec    contract.ToolSpec
	handler contract.ToolHandler
}

type MemoryAgent struct {
	// llm 是本 Agent 的模型后端，只负责带工具的一轮 Chat，不感知 session。
	llm contract.LLM
	// maxIter 是单次 Run 内 ReAct 循环的最大轮数，超出后返回 ErrMaxIterations。
	maxIter int
	// limits 是 Session 数量、单条消息和历史体积的硬上限。
	limits Limits

	// logger 承接被 recover 的 panic 详情（Error，带 session/value/stack）。
	// 只依赖 contract.Log，不依赖具体日志实现；未注入时静默跳过。
	logger contract.Log

	// store 持久化 Session 历史并提供 Run 互斥。默认内存；可注入 Redis。
	store SessionStore

	// toolsMu 保护 tools 的注册、覆盖与列表拷贝，允许与 Run 并发。
	toolsMu sync.RWMutex
	// tools 是已注册工具表；同名后写覆盖，ListTools 返回拷贝。
	tools []registeredTool
}

// NewMemoryAgent 使用默认有界资源上限创建内存 Agent。
func NewMemoryAgent(llm contract.LLM, maxIter int) *MemoryAgent {
	return NewMemoryAgentWithLimits(llm, maxIter, DefaultLimits())
}

// NewMemoryAgentWithLimits 创建内存 Agent，并显式指定有界资源上限。
// limits 中的非正值回退为默认值。
func NewMemoryAgentWithLimits(llm contract.LLM, maxIter int, limits Limits) *MemoryAgent {
	if maxIter <= 0 {
		maxIter = contract.DefaultMaxIter
	}
	limits = limits.normalize()
	return &MemoryAgent{
		llm:     llm,
		maxIter: maxIter,
		limits:  limits,
		store:   newMemoryStore(limits.MaxSessions),
	}
}

// NewHadeAgentService 从 Provider 参数创建 Agent。
// 第三、四、五个参数（Limits、contract.Log、SessionStore 或 GetClient error）均可选。
func NewHadeAgentService(params ...interface{}) (interface{}, error) {
	llm := params[0].(contract.LLM)
	maxIter := params[1].(int)
	limits := DefaultLimits()
	if len(params) > 2 {
		if configured, ok := params[2].(Limits); ok {
			limits = configured
		}
	}
	agent := NewMemoryAgentWithLimits(llm, maxIter, limits)
	if len(params) > 3 {
		if logger, ok := params[3].(contract.Log); ok {
			agent.logger = logger
		}
	}
	if len(params) > 4 && params[4] != nil {
		if err, ok := params[4].(error); ok {
			return nil, err
		}
		if store, ok := params[4].(SessionStore); ok {
			agent.store = store
		}
	}
	return agent, nil
}

// logRunPanic 把 recover 现场写进 contract.Log。日志实现 panic 时被吞掉，不影响主流程。
func (a *MemoryAgent) logRunPanic(ctx context.Context, sessionID string, recovered interface{}) {
	if a.logger == nil {
		return
	}
	defer func() { _ = recover() }()
	a.logger.Error(ctx, "agent run panic", map[string]interface{}{
		"session": sessionID,
		"value":   recovered,
		"stack":   string(debug.Stack()),
	})
}

func (a *MemoryAgent) CreateSession(ctx context.Context) (string, error) {
	return a.store.Create(ctx)
}

// GetSession 返回 Session 快照。只读 store，不会被同 Session 正在运行的 Run 阻塞。
func (a *MemoryAgent) GetSession(ctx context.Context, id string) (contract.Session, error) {
	messages, err := a.store.Open(ctx, id)
	if err != nil {
		return contract.Session{}, err
	}
	if messages == nil {
		messages = []contract.Message{}
	}
	for i := range messages {
		messages[i] = truncateMessageForRead(messages[i], contract.ContentMaxBytes)
	}
	return contract.Session{ID: id, Messages: messages}, nil
}

// RegisterTool 注册工具。名称为空或 handler 为 nil 的注册被忽略；同名注册覆盖旧实现。
func (a *MemoryAgent) RegisterTool(spec contract.ToolSpec, handler contract.ToolHandler) {
	name := strings.TrimSpace(spec.Name)
	if name == "" || handler == nil {
		return
	}
	spec.Name = name
	spec.Parameters = deepCopyParameters(spec.Parameters)

	a.toolsMu.Lock()
	defer a.toolsMu.Unlock()
	for i := range a.tools {
		if a.tools[i].spec.Name == name {
			a.tools[i] = registeredTool{spec: spec, handler: handler}
			return
		}
	}
	a.tools = append(a.tools, registeredTool{spec: spec, handler: handler})
}

func (a *MemoryAgent) ListTools() []contract.ToolSpec {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()
	out := make([]contract.ToolSpec, len(a.tools))
	for i, tool := range a.tools {
		out[i] = tool.spec
		out[i].Parameters = deepCopyParameters(tool.spec.Parameters)
	}
	return out
}

func (a *MemoryAgent) Run(
	ctx context.Context,
	sessionID, userMessage string,
	events chan<- contract.AgentEvent,
) (err error) {
	if strings.TrimSpace(userMessage) == "" {
		return contract.ErrEmptyMessage
	}
	if len(userMessage) > a.limits.MaxMessageBytes {
		return contract.ErrMessageTooLarge
	}
	session, err := a.store.TryBeginRun(ctx, sessionID)
	if err != nil {
		return err
	}
	defer session.Release()

	run := &runState{agent: a, session: session, events: events, ctx: ctx}
	run.settleDanglingToolCalls()
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		// 现场只进日志，不进事件流：客户端拿不到 panic 细节。
		a.logRunPanic(ctx, sessionID, recovered)
		// 自身逻辑异常也不能留下悬空的 assistant.tool_calls。
		run.settlePending(settleReasonInternal)
		run.tryEmit(contract.EventError, map[string]interface{}{
			"code":    "internal",
			"message": "internal error",
		})
		err = contract.ErrInternal
	}()
	return run.loop(sessionID, userMessage)
}

// execTool 调用第三方工具实现。handler 的 panic 在这里被拦截并转成 error observation，
// 既不会打崩进程，也不会中断 ReAct 循环。
func (a *MemoryAgent) execTool(ctx context.Context, name, argsJSON string) (observation string, err error) {
	handler := a.lookupTool(name)
	if handler == nil {
		return "", errors.New("unknown tool: " + name)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			observation = ""
			err = fmt.Errorf("%w: tool %s panicked: %v", contract.ErrInternal, name, recovered)
		}
	}()
	return handler(ctx, argsJSON)
}

func (a *MemoryAgent) lookupTool(name string) contract.ToolHandler {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()
	for _, tool := range a.tools {
		if tool.spec.Name == name {
			return tool.handler
		}
	}
	return nil
}

// runState 承载一轮 Run 的可变状态。
//
// pending 是"已写入历史的 assistant.tool_calls 中尚未拿到 tool 回复"的 call id。
// 任何提前退出路径都必须把 pending 补齐，否则下一轮送给 OpenAI 的历史非法。
type runState struct {
	agent   *MemoryAgent
	session RunSession
	events  chan<- contract.AgentEvent
	ctx     context.Context
	pending []string
}

func (r *runState) send(eventType string, data map[string]interface{}) bool {
	if r.events == nil {
		return true
	}
	select {
	case r.events <- contract.AgentEvent{Type: eventType, Data: data}:
		return true
	case <-r.ctx.Done():
		return false
	}
}

// tryEmit 尽力推送一个事件。它只用于收尾路径，此时调用方可能已经关闭 channel，
// 因此自带 recover：收尾逻辑绝不能再抛出第二个 panic。
func (r *runState) tryEmit(eventType string, data map[string]interface{}) {
	if r.events == nil {
		return
	}
	defer func() { _ = recover() }()
	select {
	case r.events <- contract.AgentEvent{Type: eventType, Data: data}:
	default:
	}
}

// canceled 收尾一轮被取消的 Run：先补齐未闭环的 tool_call，再尽力推一个 error 事件。
func (r *runState) canceled() error {
	r.settlePending(settleReasonCanceled)
	message := contract.ErrCanceled.Error()
	if err := r.ctx.Err(); err != nil {
		message = err.Error()
	}
	r.tryEmit(contract.EventError, map[string]interface{}{
		"code":    "canceled",
		"message": message,
	})
	return contract.ErrCanceled
}

// settlePending 为尚未完成的 tool_call 补一条 tool 消息，已完成的不会重复补。
// 配额在写入 assistant 消息时已按 settleReserveBytes 预留，所以这里不会突破历史上限。
func (r *runState) settlePending(reason string) {
	if len(r.pending) == 0 {
		return
	}
	messages := make([]contract.Message, 0, len(r.pending))
	for _, id := range r.pending {
		messages = append(messages, contract.Message{
			Role:       "tool",
			ToolCallID: id,
			Content:    truncate(reason, settleReasonMaxBytes),
		})
	}
	r.pending = nil
	r.session.AppendReserved(messages...)
}

func (r *runState) historyLimited() error {
	if !r.send(contract.EventError, map[string]interface{}{
		"code":    "history_limit",
		"message": contract.ErrHistoryLimit.Error(),
	}) {
		return r.canceled()
	}
	return contract.ErrHistoryLimit
}

func (r *runState) settleDanglingToolCalls() {
	pending := danglingToolCallIDs(r.session.Snapshot())
	if len(pending) == 0 {
		return
	}
	r.pending = pending
	r.settlePending(settleReasonInternal)
}

func danglingToolCallIDs(messages []contract.Message) []string {
	answered := map[string]struct{}{}
	for _, message := range messages {
		if message.Role == "tool" && message.ToolCallID != "" {
			answered[message.ToolCallID] = struct{}{}
		}
	}
	var pending []string
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		for _, call := range message.ToolCalls {
			if _, ok := answered[call.ID]; !ok {
				pending = append(pending, call.ID)
			}
		}
	}
	return pending
}

func (r *runState) loop(sessionID, userMessage string) error {
	limit := r.agent.limits.MaxHistoryBytes
	if err := r.session.AppendWithin(limit, 0, contract.Message{
		Role:    "user",
		Content: userMessage,
	}); err != nil {
		return contract.ErrHistoryLimit
	}
	if !r.send(contract.EventSession, map[string]interface{}{"session_id": sessionID}) {
		return r.canceled()
	}

	for i := 0; i < r.agent.maxIter; i++ {
		if r.ctx.Err() != nil {
			return r.canceled()
		}
		resp, err := r.agent.llm.Chat(r.ctx, contract.ChatRequest{
			Messages: r.session.Snapshot(),
			Tools:    r.agent.ListTools(),
		})
		if err != nil {
			if r.ctx.Err() != nil {
				return r.canceled()
			}
			if !r.send(contract.EventError, map[string]interface{}{
				"code":    "llm_failed",
				"message": err.Error(),
			}) {
				return r.canceled()
			}
			return contract.ErrLLMFailed
		}
		if !r.send(contract.EventThought, map[string]interface{}{
			"content": truncate(resp.Message.Content, contract.ContentMaxBytes),
		}) {
			return r.canceled()
		}

		if len(resp.Message.ToolCalls) > 0 {
			if err := r.runToolCalls(resp.Message.Content, resp.Message.ToolCalls); err != nil {
				return err
			}
			continue
		}

		if err := r.session.AppendWithin(limit, 0, contract.Message{
			Role:    "assistant",
			Content: resp.Message.Content,
		}); err != nil {
			return r.historyLimited()
		}
		if !r.send(contract.EventFinal, map[string]interface{}{
			"content": truncate(resp.Message.Content, contract.ContentMaxBytes),
		}) {
			return r.canceled()
		}
		return nil
	}

	if !r.send(contract.EventError, map[string]interface{}{
		"code":    "max_iterations",
		"message": "max iterations reached",
	}) {
		return r.canceled()
	}
	return contract.ErrMaxIterations
}

func (r *runState) runToolCalls(thought string, calls []contract.ToolCall) error {
	limit := r.agent.limits.MaxHistoryBytes
	mark := r.session.Length()
	assistant := contract.Message{Role: "assistant", Content: thought, ToolCalls: calls}
	if err := r.session.AppendWithin(limit, settleReserveBytes(calls), assistant); err != nil {
		// assistant 尚未入历史，直接拒绝即可，历史仍然合法。
		return r.historyLimited()
	}
	r.pending = make([]string, 0, len(calls))
	for _, call := range calls {
		r.pending = append(r.pending, call.ID)
	}

	for index, call := range calls {
		arguments := map[string]interface{}{}
		argumentsValid := json.Unmarshal([]byte(call.Arguments), &arguments) == nil
		if !argumentsValid {
			arguments = map[string]interface{}{}
		}
		if !r.send(contract.EventAction, map[string]interface{}{
			"name":      call.Name,
			"arguments": arguments,
		}) {
			return r.canceled()
		}

		observation, execErr := r.agent.execTool(r.ctx, call.Name, call.Arguments)
		if execErr != nil {
			observation = execErr.Error()
		}
		observation = truncate(observation, contract.ContentMaxBytes)
		if !argumentsValid {
			observation = truncate(
				"invalid tool arguments: "+call.Arguments+" ; "+observation,
				contract.ContentMaxBytes,
			)
		}

		// 真实结果先落历史再推事件：即使紧接着被取消，这个 call 也已经闭环。
		result := contract.Message{Role: "tool", ToolCallID: call.ID, Content: observation}
		if err := r.session.AppendWithin(limit, settleReserveBytes(calls[index+1:]), result); err != nil {
			// 回滚整组 assistant.tool_calls，历史既不超限也不残留悬空 call。
			r.session.TruncateTo(mark)
			r.pending = nil
			return r.historyLimited()
		}
		r.pending = r.pending[1:]

		if !r.send(contract.EventObservation, map[string]interface{}{
			"name":    call.Name,
			"content": observation,
		}) {
			return r.canceled()
		}
	}
	return nil
}
