package agent

import (
	"context"
	"strings"
	"time"

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
	return []interface{}{llm, maxIter, logger, sessionStoreFrom(c, logger)}
}

func wantRedisSessionStore(c framework.Container) bool {
	if !c.IsBind(contract.ConfigKey) {
		return false
	}
	cfg := c.MustMake(contract.ConfigKey).(contract.Config)
	return strings.EqualFold(strings.TrimSpace(cfg.GetString("agent.session_store")), "redis")
}

func logSessionStoreFallback(logger contract.Log, reason string) {
	if logger == nil {
		return
	}
	defer func() { _ = recover() }()
	logger.Warn(context.Background(), "agent session store fallback to memory", map[string]interface{}{
		"reason": reason,
	})
}

// sessionStoreFrom 默认内存。仅当 agent.session_store=redis 且 Redis Ping 成功时才用 Redis。
func sessionStoreFrom(c framework.Container, logger contract.Log) interface{} {
	if !wantRedisSessionStore(c) {
		return nil
	}
	if !c.IsBind(contract.RedisKey) {
		logSessionStoreFallback(logger, "redis service not bound")
		return nil
	}
	redisService := c.MustMake(contract.RedisKey).(contract.RedisService)
	client, err := redisService.GetClient()
	if err != nil {
		logSessionStoreFallback(logger, err.Error())
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logSessionStoreFallback(logger, err.Error())
		return nil
	}
	return newRedisStore(client)
}
