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
	limits := DefaultLimits()
	if c.IsBind(contract.ConfigKey) {
		cfg := c.MustMake(contract.ConfigKey).(contract.Config)
		if cfg.IsExist("agent.max_iterations") {
			maxIter = cfg.GetInt("agent.max_iterations")
		}
		limits = limitsFromConfig(cfg)
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
			store = newRedisStore(client, limits.MaxSessions)
		}
	}
	return []interface{}{llm, maxIter, limits, logger, store}
}

// limitsFromConfig 从 agent.yaml 读取有界资源上限。
// 缺失或非正的键保持默认值，因此配置文件不写这些键时行为完全不变。
func limitsFromConfig(cfg contract.Config) Limits {
	limits := DefaultLimits()
	if cfg.IsExist("agent.max_sessions") {
		limits.MaxSessions = cfg.GetInt("agent.max_sessions")
	}
	if cfg.IsExist("agent.max_message_bytes") {
		limits.MaxMessageBytes = cfg.GetInt("agent.max_message_bytes")
	}
	if cfg.IsExist("agent.max_history_bytes") {
		limits.MaxHistoryBytes = cfg.GetInt("agent.max_history_bytes")
	}
	return limits.normalize()
}
