package http

import (
	"github.com/gohade/hade/app/http/job"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
)

// NewHttpEngine 创建了一个绑定了路由的Web引擎
func NewHttpEngine(container framework.Container) (*gin.Engine, error) {
	// 设置为Release，为的是默认在启动中不输出调试信息
	gin.SetMode(gin.ReleaseMode)
	// 默认启动一个Web引擎
	r := gin.New()

	// 注册各种job
	RegisterJobs(container)

	// 设置了Engine
	r.SetContainer(container)

	// 默认注册recovery中间件
	r.Use(gin.Recovery())

	// 业务绑定路由操作
	Routes(r)
	// 返回绑定路由后的Web引擎
	return r, nil
}

// RegisterJobs 注册各种Job
func RegisterJobs(container framework.Container) {
	queue := container.MustMake(contract.QueueKey).(contract.QueueService)
	queue.Register(&job.CreateUser{})
}
