package agent

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sse"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// Handler 处理 Agent Session 与消息流请求。
type Handler struct{}

// CreateSession 创建一个内存 Session。
func (*Handler) CreateSession(c *gin.Context) {
	id, err := agentFrom(c).CreateSession(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GetSession 查询 Session 快照。
func (*Handler) GetSession(c *gin.Context) {
	session, err := agentFrom(c).GetSession(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// Messages 执行一次 Agent Run，并将业务事件转为 SSE。
func (*Handler) Messages(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	sessionID := c.Param("id")
	agent := agentFrom(c)
	events := make(chan contract.AgentEvent, 64)
	result := make(chan error, 1)
	go func() {
		result <- agent.Run(ctx, sessionID, request.Message, events)
		close(events)
	}()

	first, runErr, hasFirst, hasResult := awaitFirstEvent(ctx, events, result)
	if !hasFirst {
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
	writeSSE(c, &sequence, first)
	for event := range events {
		writeSSE(c, &sequence, event)
	}
	if !hasResult {
		runErr = <-result
	}
	_ = runErr // Run 已发送业务 error 事件；handler 只负责补唯一 done。
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

func writeAgentError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, contract.ErrEmptyMessage):
		status = http.StatusBadRequest
	case errors.Is(err, contract.ErrSessionNotFound):
		status = http.StatusNotFound
	case errors.Is(err, contract.ErrSessionBusy):
		status = http.StatusConflict
	}
	message := http.StatusText(status)
	if err != nil {
		message = err.Error()
	}
	c.JSON(status, gin.H{"error": message})
}

func agentFrom(c *gin.Context) contract.Agent {
	return c.MustMake(contract.AgentKey).(contract.Agent)
}
