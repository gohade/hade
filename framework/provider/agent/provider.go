package agent

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeAgentProvider struct{}

func (p *HadeAgentProvider) Name() string { return contract.AgentKey }

func (p *HadeAgentProvider) Register(c framework.Container) framework.NewInstance {
	return NewHadeAgentService
}

func (p *HadeAgentProvider) Boot(c framework.Container) error { return nil }

func (p *HadeAgentProvider) IsDefer() bool { return true }

func (p *HadeAgentProvider) Params(c framework.Container) []interface{} {
	llm := c.MustMake(contract.LLMKey).(contract.LLM)
	maxIter := contract.DefaultMaxIter
	if c.IsBind(contract.ConfigKey) {
		cfg := c.MustMake(contract.ConfigKey).(contract.Config)
		if cfg.IsExist("agent.max_iterations") {
			maxIter = cfg.GetInt("agent.max_iterations")
		}
	}
	var logger contract.Log
	if c.IsBind(contract.LogKey) {
		logger = c.MustMake(contract.LogKey).(contract.Log)
	}
	var store interface{}
	if c.IsBind(contract.RedisKey) {
		redisService := c.MustMake(contract.RedisKey).(contract.RedisService)
		client, err := redisService.GetClient()
		if err != nil {
			store = err
		} else {
			store = newRedisStore(client)
		}
	}
	return []interface{}{llm, maxIter, logger, store}
}
