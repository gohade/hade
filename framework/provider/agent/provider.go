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
	return []interface{}{llm, maxIter}
}
