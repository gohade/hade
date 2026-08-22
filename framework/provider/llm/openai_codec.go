package llm

import (
	"encoding/json"
	"fmt"

	"github.com/gohade/hade/framework/contract"
)

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openAIToolCallFunction `json:"function"`
}

type openAIToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	FinishReason string                `json:"finish_reason"`
	Message      openAIResponseMessage `json:"message"`
}

type openAIResponseMessage struct {
	Role      string           `json:"role"`
	Content   *string          `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

// BuildOpenAIRequest 将通用 LLM 请求编码为 OpenAI Chat Completions 请求。
func BuildOpenAIRequest(req contract.ChatRequest, model string) ([]byte, error) {
	openReq := openAIRequest{
		Model:       model,
		Messages:    make([]openAIMessage, 0, len(req.Messages)),
		Tools:       make([]openAITool, 0, len(req.Tools)),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	for _, message := range req.Messages {
		openMessage := openAIMessage{
			Role:       message.Role,
			Content:    message.Content,
			ToolCallID: message.ToolCallID,
			ToolCalls:  make([]openAIToolCall, 0, len(message.ToolCalls)),
		}
		for _, call := range message.ToolCalls {
			openMessage.ToolCalls = append(openMessage.ToolCalls, openAIToolCall{
				ID:   call.ID,
				Type: "function",
				Function: openAIToolCallFunction{
					Name:      call.Name,
					Arguments: call.Arguments,
				},
			})
		}
		openReq.Messages = append(openReq.Messages, openMessage)
	}
	for _, tool := range req.Tools {
		openReq.Tools = append(openReq.Tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return json.Marshal(openReq)
}

// ParseOpenAIResponse 将 OpenAI Chat Completions 响应解码为通用 LLM 响应。
func ParseOpenAIResponse(data []byte) (contract.ChatResponse, error) {
	var openResp openAIResponse
	if err := json.Unmarshal(data, &openResp); err != nil {
		return contract.ChatResponse{}, fmt.Errorf("decode OpenAI response: %w", err)
	}
	if len(openResp.Choices) == 0 {
		return contract.ChatResponse{}, fmt.Errorf("OpenAI response has no choices")
	}

	choice := openResp.Choices[0]
	message := contract.Message{Role: choice.Message.Role}
	if choice.Message.Content != nil {
		message.Content = *choice.Message.Content
	}
	toolCalls := make([]contract.ToolCall, 0, len(choice.Message.ToolCalls))
	for _, call := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, contract.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}
	message.ToolCalls = append([]contract.ToolCall(nil), toolCalls...)

	return contract.ChatResponse{
		Message:   message,
		ToolCalls: toolCalls,
		Finish:    choice.FinishReason,
	}, nil
}
