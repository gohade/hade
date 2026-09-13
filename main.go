// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	agentapp "github.com/gohade/hade/app/agent"
	"github.com/gohade/hade/app/console"
	"github.com/gohade/hade/app/grpc"
	"github.com/gohade/hade/app/http"
	"github.com/gohade/hade/framework"
	agentprovider "github.com/gohade/hade/framework/provider/agent"
	"github.com/gohade/hade/framework/provider/app"
	"github.com/gohade/hade/framework/provider/cache"
	"github.com/gohade/hade/framework/provider/config"
	"github.com/gohade/hade/framework/provider/distributed"
	"github.com/gohade/hade/framework/provider/env"
	"github.com/gohade/hade/framework/provider/id"
	"github.com/gohade/hade/framework/provider/kernel"
	llmprovider "github.com/gohade/hade/framework/provider/llm"
	"github.com/gohade/hade/framework/provider/log"
	"github.com/gohade/hade/framework/provider/orm"
	"github.com/gohade/hade/framework/provider/redis"
	"github.com/gohade/hade/framework/provider/sls"
	"github.com/gohade/hade/framework/provider/ssh"
	"github.com/gohade/hade/framework/provider/trace"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// 初始化服务容器
	container := framework.NewHadeContainer()
	// 绑定App服务提供者
	_ = container.Bind(&app.HadeAppProvider{})
	// 后续初始化需要绑定的服务提供者...
	_ = container.Bind(&env.HadeEnvProvider{})
	_ = container.Bind(&distributed.LocalDistributedProvider{})
	if err := container.Bind(&config.HadeConfigProvider{}); err != nil {
		return err
	}
	if err := container.Bind(&llmprovider.HadeLLMProvider{}); err != nil {
		return err
	}
	if err := container.Bind(&agentprovider.HadeAgentProvider{}); err != nil {
		return err
	}
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
	httpEngine, err := http.NewHttpEngine(container)
	if err != nil {
		return err
	}
	kernelProvider.HttpEngine = httpEngine

	grpcEngine, err := grpc.NewGrpcEngine(container)
	if err != nil {
		return err
	}
	kernelProvider.GrpcEngine = grpcEngine

	agentEngine, err := agentapp.NewAgentEngine(container)
	if err != nil {
		return err
	}
	kernelProvider.AgentEngine = agentEngine

	if err := container.Bind(kernelProvider); err != nil {
		return err
	}

	// 运行root命令
	return console.RunCommand(container)
}
