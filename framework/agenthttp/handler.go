package agenthttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/gin-contrib/sse"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	pkgerrors "github.com/pkg/errors"
)

const agentUnavailable = "agent service unavailable"

func agentFrom(c *gin.Context) (contract.Agent, error) {
	instance, err := c.Make(contract.AgentKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[agent] make: %v\n", err)
		return nil, pkgerrors.New(agentUnavailable)
	}
	typed, ok := instance.(contract.Agent)
	if !ok || typed == nil {
		fmt.Fprintf(os.Stderr, "[agent] service %s is not a contract.Agent\n", contract.AgentKey)
		return nil, pkgerrors.New(agentUnavailable)
	}
	return typed, nil
}

func writeUnavailable(c *gin.Context, _ error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": agentUnavailable})
}

// CreateSession 创建一个内存 Session。
func CreateSession(c *gin.Context) {
	agent, err := agentFrom(c)
	if err != nil {
		writeUnavailable(c, err)
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
func GetSession(c *gin.Context) {
	agent, err := agentFrom(c)
	if err != nil {
		writeUnavailable(c, err)
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
func Messages(c *gin.Context) {
	agent, err := agentFrom(c)
	if err != nil {
		writeUnavailable(c, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, contract.RequestBodyMaxBytes)
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
		defer safeClose(events)
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "[agent] run panic: %v\n%s\n", recovered, debug.Stack())
				result <- contract.ErrInternal
			}
		}()
		result <- agent.Run(ctx, sessionID, request.Message, events)
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

func safeClose(events chan contract.AgentEvent) {
	defer func() { _ = recover() }()
	close(events)
}

func errorEventData(err error) map[string]interface{} {
	code := "internal"
	switch {
	case errors.Is(err, contract.ErrCanceled):
		code = "canceled"
	case errors.Is(err, contract.ErrLLMFailed):
		code = "llm_failed"
	case errors.Is(err, contract.ErrMaxIterations):
		code = "max_iterations"
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

func bindErrorStatus(err error) int {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
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
	}
	message := http.StatusText(status)
	if err != nil {
		message = err.Error()
	}
	c.JSON(status, gin.H{"error": message})
}
