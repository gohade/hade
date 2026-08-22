package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gohade/hade/framework/contract"
)

const (
	openAIChatCompletionsPath = "/chat/completions"
	openAIResponseLimit       = 10 << 20
)

var _ contract.LLM = (*OpenAI)(nil)

// OpenAI 是 OpenAI Chat Completions API 的兼容客户端。
type OpenAI struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAI 创建一个使用 60 秒超时的 OpenAI 兼容客户端。
func NewOpenAI(baseURL, apiKey, model string) *OpenAI {
	return &OpenAI{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// NewHadeLLM 使用 Provider 参数创建 LLM 服务。
func NewHadeLLM(params ...interface{}) (interface{}, error) {
	baseURL := params[0].(string)
	apiKey := params[1].(string)
	model := params[2].(string)
	return NewOpenAI(baseURL, apiKey, model), nil
}

// Chat 调用 OpenAI 兼容的 Chat Completions API。
func (o *OpenAI) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	if strings.TrimSpace(o.apiKey) == "" {
		return contract.ChatResponse{}, fmt.Errorf("%w: api key is empty", contract.ErrLLMFailed)
	}

	body, err := BuildOpenAIRequest(req, o.model)
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: encode request", contract.ErrLLMFailed)
	}
	url := strings.TrimRight(o.baseURL, "/") + openAIChatCompletionsPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: create request", contract.ErrLLMFailed)
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: request failed", contract.ErrLLMFailed)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return contract.ChatResponse{}, fmt.Errorf("%w: upstream status %d", contract.ErrLLMFailed, resp.StatusCode)
	}

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, openAIResponseLimit))
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: read response", contract.ErrLLMFailed)
	}
	result, err := ParseOpenAIResponse(responseBody)
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: parse response", contract.ErrLLMFailed)
	}
	return result, nil
}
