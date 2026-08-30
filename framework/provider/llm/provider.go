package llm

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

// HadeLLMProvider 注册 OpenAI 兼容的 LLM 服务。
type HadeLLMProvider struct{}

func (p *HadeLLMProvider) Name() string {
	return contract.LLMKey
}

func (p *HadeLLMProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeLLM
}

func (p *HadeLLMProvider) Boot(c framework.Container) error {
	return nil
}

func (p *HadeLLMProvider) IsDefer() bool {
	return true
}

func (p *HadeLLMProvider) Params(c framework.Container) []interface{} {
	baseURL := defaultOpenAIBaseURL
	apiKey := ""
	model := defaultOpenAIModel
	if c.IsBind(contract.ConfigKey) {
		config := c.MustMake(contract.ConfigKey).(contract.Config)
		if config.IsExist("llm.base_url") {
			baseURL = config.GetString("llm.base_url")
		}
		if config.IsExist("llm.api_key") {
			apiKey = config.GetString("llm.api_key")
		}
		if config.IsExist("llm.model") {
			model = config.GetString("llm.model")
		}
	}
	var logger contract.Log
	if c.IsBind(contract.LogKey) {
		logger = c.MustMake(contract.LogKey).(contract.Log)
	}
	return []interface{}{baseURL, apiKey, model, logger}
}
