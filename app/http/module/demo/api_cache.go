package demo

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/redis"
	"time"
)

// DemoRedis redis的路由方法
func (api *DemoApi) DemoRedis(c *gin.Context) {
	logger := c.MustMakeLog()
	logger.Info(c, "request start", nil)

	// 初始化一个redis
	redisService := c.MustMake(contract.RedisKey).(contract.RedisService)
	client, err := redisService.GetClient(redis.WithConfigPath("cache.default"), redis.WithRedisConfig(func(options *contract.RedisConfig) {
		options.MaxRetries = 3
	}))
	if err != nil {
		logger.Error(c, err.Error(), nil)
		c.AbortWithError(50001, err)
		return
	}
	if err := client.Set(c, "foo", "bar", 1*time.Hour).Err(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	val := client.Get(c, "foo").String()
	logger.Info(c, "redis get", map[string]interface{}{
		"val": val,
	})

	if err := client.Del(c, "foo").Err(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, "ok")
}

// DemoCache cache的简单例子
func (api *DemoApi) DemoCache(c *gin.Context) {
	logger := c.MustMakeLog()
	logger.Info(c, "request start", nil)
	// 初始化cache服务
	cacheService := c.MustMake(contract.CacheKey).(contract.CacheService)
	// 设置key为foo
	err := cacheService.Set(c, "foo", "bar", 1*time.Hour)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	// 获取key为foo
	val, err := cacheService.Get(c, "foo")
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	logger.Info(c, "cache get", map[string]interface{}{
		"val": val,
	})
	// 删除key为foo
	if err := cacheService.Del(c, "foo"); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.JSON(200, "ok")
}
