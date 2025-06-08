// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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
	hadelog "github.com/gohade/hade/framework/provider/log"
	"github.com/gohade/hade/framework/provider/orm"
	"github.com/gohade/hade/framework/provider/redis"
	"github.com/gohade/hade/framework/provider/sls"
	"github.com/gohade/hade/framework/provider/ssh"
	"github.com/gohade/hade/framework/provider/trace"
)

func main() {
	// 初始化服务容器
	container := framework.NewHadeContainer()

	// 绑定基础服务提供者（这些是必须的）
	if err := container.Bind(&app.HadeAppProvider{}); err != nil {
		log.Fatalf("Failed to bind App provider: %v", err)
	}
	if err := container.Bind(&env.HadeEnvProvider{}); err != nil {
		log.Fatalf("Failed to bind Env provider: %v", err)
	}
	if err := container.Bind(&config.HadeConfigProvider{}); err != nil {
		log.Fatalf("Failed to bind Config provider: %v", err)
	}

	// 绑定核心服务提供者
	if err := container.Bind(&hadelog.HadeLogServiceProvider{}); err != nil {
		log.Fatalf("Failed to bind Log provider: %v", err)
	}
	if err := container.Bind(&id.HadeIDProvider{}); err != nil {
		log.Printf("Warning: Failed to bind ID provider: %v", err)
	}
	if err := container.Bind(&trace.HadeTraceProvider{}); err != nil {
		log.Printf("Warning: Failed to bind Trace provider: %v", err)
	}

	// 绑定可选服务提供者（失败不影响启动）
	if err := container.Bind(&distributed.LocalDistributedProvider{}); err != nil {
		log.Printf("Warning: Failed to bind Distributed provider: %v", err)
	}
	if err := container.Bind(&orm.GormProvider{}); err != nil {
		log.Printf("Warning: Failed to bind ORM provider: %v", err)
	}
	if err := container.Bind(&redis.RedisProvider{}); err != nil {
		log.Printf("Warning: Failed to bind Redis provider: %v", err)
	}
	if err := container.Bind(&cache.HadeCacheProvider{}); err != nil {
		log.Printf("Warning: Failed to bind Cache provider: %v", err)
	}
	if err := container.Bind(&ssh.SSHProvider{}); err != nil {
		log.Printf("Warning: Failed to bind SSH provider: %v", err)
	}
	if err := container.Bind(&sls.HadeSLSProvider{}); err != nil {
		log.Printf("Warning: Failed to bind SLS provider: %v", err)
	}

	// 将HTTP和grpc引擎初始化,并且作为服务提供者绑定到服务容器中
	kernelProvider := &kernel.HadeKernelProvider{}
	if engine, err := http.NewHttpEngine(container); err == nil {
		kernelProvider.HttpEngine = engine
		log.Println("HTTP engine initialized successfully")
	} else {
		log.Printf("Warning: Failed to initialize HTTP engine: %v", err)
	}

	if engine, err := grpc.NewGrpcEngine(container); err == nil {
		kernelProvider.GrpcEngine = engine
		log.Println("gRPC engine initialized successfully")
	} else {
		log.Printf("Warning: Failed to initialize gRPC engine: %v", err)
	}

	if err := container.Bind(kernelProvider); err != nil {
		log.Fatalf("Failed to bind Kernel provider: %v", err)
	}

	// 设置优雅退出
	go handleSignals()

	// 运行root命令
	if err := console.RunCommand(container); err != nil {
		log.Fatalf("Failed to run command: %v", err)
	}
}

// handleSignals 处理系统信号
func handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down...", sig)

	// TODO: 在这里添加清理逻辑
	// - 关闭 HTTP 服务器
	// - 关闭数据库连接
	// - 关闭 Redis 连接
	// - 其他清理工作

	os.Exit(0)
}
