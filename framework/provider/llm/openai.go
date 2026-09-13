package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gohade/hade/framework/contract"
)

const (
	openAIChatCompletionsPath = "/chat/completions"
	openAIResponseLimit       = 10 << 20
)

var _ contract.LLM = (*OpenAI)(nil)
var _ contract.ConfigurableLLM = (*OpenAI)(nil)

// OpenAI 是 OpenAI Chat Completions API 的兼容客户端。
type OpenAI struct {
	mu      sync.RWMutex
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	logger  contract.Log
}

// Configuration 返回当前连接配置的快照。
func (o *OpenAI) Configuration() contract.LLMConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return contract.LLMConfig{BaseURL: o.baseURL, APIKey: o.apiKey, Model: o.model}
}

// Configure 原子替换后续 Chat 使用的模型配置，已在执行中的请求不受影响。
func (o *OpenAI) Configure(config contract.LLMConfig) error {
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	config.Model = strings.TrimSpace(config.Model)
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.BaseURL == "" {
		return errors.New("base_url is required")
	}
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("base_url must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return errors.New("base_url must not contain credentials")
	}
	if config.Model == "" {
		return errors.New("model is required")
	}
	if len(config.BaseURL) > 2048 || len(config.Model) > 256 || len(config.APIKey) > 16<<10 {
		return errors.New("configuration value is too long")
	}

	o.mu.Lock()
	o.baseURL = strings.TrimRight(config.BaseURL, "/")
	o.apiKey = config.APIKey
	o.model = config.Model
	o.mu.Unlock()
	return nil
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
	client := NewOpenAI(baseURL, apiKey, model)
	if len(params) > 3 {
		if logger, ok := params[3].(contract.Log); ok {
			client.logger = logger
		}
	}
	return client, nil
}

// Chat 调用 OpenAI 兼容的 Chat Completions API。
func (o *OpenAI) Chat(ctx context.Context, req contract.ChatRequest) (contract.ChatResponse, error) {
	config := o.Configuration()
	if strings.TrimSpace(config.APIKey) == "" {
		return contract.ChatResponse{}, fmt.Errorf("%w: api key is empty", contract.ErrLLMFailed)
	}

	body, err := BuildOpenAIRequest(req, config.Model)
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: encode request", contract.ErrLLMFailed)
	}
	endpoint := strings.TrimRight(config.BaseURL, "/") + openAIChatCompletionsPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: create request", contract.ErrLLMFailed)
	}
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)
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
	o.logChat(ctx, config.Model, body, responseBody)
	result, err := ParseOpenAIResponse(responseBody)
	if err != nil {
		return contract.ChatResponse{}, fmt.Errorf("%w: parse response", contract.ErrLLMFailed)
	}
	return result, nil
}

func (o *OpenAI) logChat(ctx context.Context, model string, requestBody, responseBody []byte) {
	if o.logger == nil {
		return
	}
	defer func() { _ = recover() }()
	o.logger.Debug(ctx, "llm chat", map[string]interface{}{
		"model":    model,
		"request":  string(requestBody),
		"response": string(responseBody),
	})
}
