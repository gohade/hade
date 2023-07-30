// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package main

import (
	"github.com/gohade/hade/app/console"
	"github.com/gohade/hade/app/grpc"
	"github.com/gohade/hade/app/http"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/provider/app"
	"github.com/gohade/hade/framework/provider/cache"
	"github.com/gohade/hade/framework/provider/config"
	"github.com/gohade/hade/framework/provider/distributed"
	"github.com/gohade/hade/framework/provider/env"
	"github.com/gohade/hade/framework/provider/id"
	"github.com/gohade/hade/framework/provider/kernel"
	"github.com/gohade/hade/framework/provider/log"
	"github.com/gohade/hade/framework/provider/orm"
	"github.com/gohade/hade/framework/provider/redis"
	"github.com/gohade/hade/framework/provider/sls"
	"github.com/gohade/hade/framework/provider/ssh"
	"github.com/gohade/hade/framework/provider/trace"
)

func main() {
	// 初始化服务容器
	container := framework.NewHadeContainer()
	// 绑定App服务提供者
	_ = container.Bind(&app.HadeAppProvider{})
	// 后续初始化需要绑定的服务提供者...
	_ = container.Bind(&env.HadeEnvProvider{})
	_ = container.Bind(&distributed.LocalDistributedProvider{})
	_ = container.Bind(&config.HadeConfigProvider{})
	_ = container.Bind(&id.HadeIDProvider{})
	_ = container.Bind(&trace.HadeTraceProvider{})
	_ = container.Bind(&log.HadeLogServiceProvider{})
	_ = container.Bind(&orm.GormProvider{})
	_ = container.Bind(&redis.RedisProvider{})
	_ = container.Bind(&cache.HadeCacheProvider{})
	_ = container.Bind(&ssh.SSHProvider{})
	_ = container.Bind(&sls.HadeSLSProvider{})

	// 将HTTP和grpc引擎初始化,并且作为服务提供者绑定到服务容器中
	kernelProvider := &kernel.HadeKernelProvider{}
	if engine, err := http.NewHttpEngine(container); err == nil {
		kernelProvider.HttpEngine = engine
	}

	if engine, err := grpc.NewGrpcEngine(container); err == nil {
		kernelProvider.GrpcEngine = engine
	}
	_ = container.Bind(kernelProvider)

	// 运行root命令
	_ = console.RunCommand(container)
}
