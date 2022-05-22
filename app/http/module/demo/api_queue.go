package demo

import (
	"github.com/gohade/hade/app/http/job"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// DemoQueue 队列
func (api *DemoApi) DemoQueue(c *gin.Context) {
	logger := c.MustMakeLog()

	queue := c.MustMake(contract.QueueKey).(contract.QueueService)
	if err := queue.Push(c, &job.CreateUser{Name: "foo"}); err != nil {
		logger.Error(c, "push job error", map[string]interface{}{
			"error": err,
		})
	}

	c.JSON(200, "ok")
}
