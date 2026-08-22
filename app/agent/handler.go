package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/gin-contrib/sse"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// Handler 处理 Agent Session 与消息流请求。
// Agent 实例在首个请求时按需从容器解析，构造 Engine 时不实例化任何 Provider。
type Handler struct {
	resolver *agentResolver
}

// NewHandler 创建带惰性 Agent 解析器的 Handler。
func NewHandler() *Handler {
	return &Handler{resolver: &agentResolver{}}
}

// CreateSession 创建一个内存 Session。
func (h *Handler) CreateSession(c *gin.Context) {
	agent, err := h.resolver.resolve(c)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	id, err := agent.CreateSession(c.Request.Context())
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GetSession 查询 Session 快照。
func (h *Handler) GetSession(c *gin.Context) {
	agent, err := h.resolver.resolve(c)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	session, err := agent.GetSession(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// Messages 执行一次 Agent Run，并将业务事件转为 SSE。
func (h *Handler) Messages(c *gin.Context) {
	agent, err := h.resolver.resolve(c)
	if err != nil {
		writeAgentError(c, err)
		return
	}

	c.Request.Body = newLimitedBody(c.Request.Body, contract.RequestBodyMaxBytes)
	var request struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(bindErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	sessionID := c.Param("id")
	events := make(chan contract.AgentEvent, 64)
	result := make(chan error, 1)
	go func() {
		// defer 逆序执行：先由 recover 写入唯一一次 result，再关闭 events。
		// 这样第三方 contract.Agent 实现的 Run panic 既不会打崩进程，
		// 也不会造成 events 重复关闭或往已关闭 channel 发送。
		defer safeClose(events)
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "[agent] run panic: %v\n%s\n", recovered, debug.Stack())
				// 对外只暴露 sentinel，panic 细节不进响应体。
				result <- contract.ErrInternal
			}
		}()
		result <- agent.Run(ctx, sessionID, request.Message, events)
	}()

	first, runErr, hasFirst, hasResult := awaitFirstEvent(ctx, events, result)
	if !hasFirst {
		// 尚未升流，包括 Run 直接 panic 的场景，一律回普通 JSON 错误。
		if hasResult {
			writeAgentError(c, runErr)
		}
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	sequence := 0
	sawError := first.Type == contract.EventError
	writeSSE(c, &sequence, first)
	for event := range events {
		if event.Type == contract.EventError {
			sawError = true
		}
		writeSSE(c, &sequence, event)
	}
	if !hasResult {
		runErr = <-result
	}
	// Run 以错误结束但一个 error 事件都没发过时（第三方实现 panic、提前 return 等），
	// 这里按错误类型补且只补一条，保证"流里出现过 error"与"Run 失败"始终一致。
	if runErr != nil && !sawError {
		writeSSE(c, &sequence, contract.AgentEvent{
			Type: contract.EventError,
			Data: errorEventData(runErr),
		})
	}
	writeSSE(c, &sequence, contract.AgentEvent{
		Type: contract.EventDone,
		Data: map[string]interface{}{},
	})
}

func awaitFirstEvent(
	ctx context.Context,
	events <-chan contract.AgentEvent,
	result <-chan error,
) (contract.AgentEvent, error, bool, bool) {
	select {
	case event, ok := <-events:
		if ok {
			return event, nil, true, false
		}
		return contract.AgentEvent{}, <-result, false, true
	case err := <-result:
		// Run 可能已快速写完缓冲事件；事件优先于其返回结果。
		select {
		case event, ok := <-events:
			if ok {
				return event, err, true, true
			}
		default:
		}
		return contract.AgentEvent{}, err, false, true
	case <-ctx.Done():
		return contract.AgentEvent{}, ctx.Err(), false, false
	}
}

func writeSSE(c *gin.Context, sequence *int, event contract.AgentEvent) {
	*sequence = *sequence + 1
	c.Render(-1, sse.Event{
		Event: event.Type,
		Id:    strconv.Itoa(*sequence),
		Data:  event.Data,
	})
	c.Writer.Flush()
}

// safeClose 关闭事件 channel。第三方 Agent 实现可能自己关过一次，
// 重复 close 只应被吞掉，不能打崩进程。
func safeClose(events chan contract.AgentEvent) {
	defer func() { _ = recover() }()
	close(events)
}

// errorEventData 把 Run 的返回错误映射成 error 事件载荷。
// message 一律用 sentinel 文案，不携带内部细节。
func errorEventData(err error) map[string]interface{} {
	code := "internal"
	switch {
	case errors.Is(err, contract.ErrCanceled):
		code = "canceled"
	case errors.Is(err, contract.ErrLLMFailed):
		code = "llm_failed"
	case errors.Is(err, contract.ErrMaxIterations):
		code = "max_iterations"
	case errors.Is(err, contract.ErrHistoryLimit):
		code = "history_limit"
	case errors.Is(err, contract.ErrMessageTooLarge):
		code = "message_too_large"
	case errors.Is(err, contract.ErrSessionBusy):
		code = "session_busy"
	case errors.Is(err, contract.ErrSessionNotFound):
		code = "session_not_found"
	case errors.Is(err, contract.ErrEmptyMessage):
		code = "empty_message"
	}
	message := "internal error"
	if code != "internal" {
		message = code
	}
	return map[string]interface{}{"code": code, "message": message}
}

// errBodyTooLarge 是请求体超限的哨兵错误。
//
// 这里不用 http.MaxBytesReader / http.MaxBytesError：后者是 Go 1.19 才有的类型，
// 而本仓库的 go 指令是 1.18。自带受限读取器可以在 1.18 上编译，并保留可 errors.Is
// 判定的哨兵语义。
var errBodyTooLarge = errors.New("request body too large")

// limitedBody 在读满 limit+1 字节时返回 errBodyTooLarge。
// 多读一个字节是为了区分"刚好等于上限"和"超过上限"。
type limitedBody struct {
	reader    io.ReadCloser
	remaining int64
}

func newLimitedBody(body io.ReadCloser, limit int64) *limitedBody {
	return &limitedBody{reader: body, remaining: limit + 1}
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, errBodyTooLarge
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.reader.Read(p)
	b.remaining -= int64(n)
	if b.remaining <= 0 {
		// 丢弃越界那一段，避免解析器凑出一个"看起来完整"的值后忽略错误。
		return 0, errBodyTooLarge
	}
	return n, err
}

func (b *limitedBody) Close() error {
	if b.reader == nil {
		return nil
	}
	return b.reader.Close()
}

// bindErrorStatus 区分请求体超限与普通解析失败。
func bindErrorStatus(err error) int {
	if errors.Is(err, errBodyTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func writeAgentError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, contract.ErrEmptyMessage):
		status = http.StatusBadRequest
	case errors.Is(err, contract.ErrSessionNotFound):
		status = http.StatusNotFound
	case errors.Is(err, contract.ErrSessionBusy):
		status = http.StatusConflict
	case errors.Is(err, contract.ErrMessageTooLarge), errors.Is(err, contract.ErrHistoryLimit):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, contract.ErrSessionLimit):
		// 容量耗尽属于服务端资源状态，而非单客户端限速，用 503 表达。
		status = http.StatusServiceUnavailable
	}
	message := http.StatusText(status)
	if err != nil {
		message = err.Error()
	}
	c.JSON(status, gin.H{"error": message})
}
