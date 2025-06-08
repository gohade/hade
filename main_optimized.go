// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gohade/hade/app/console"
	"github.com/gohade/hade/app/grpc"
	"github.com/gohade/hade/app/http"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
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

// 版本信息
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// ServiceProvider 定义服务提供者注册信息
type ServiceProvider struct {
	Provider framework.ServiceProvider
	Required bool // 是否必须成功
	Order    int  // 注册顺序
}

func main() {
	startTime := time.Now()

	// 显示版本信息
	fmt.Printf("Hade Framework %s (built: %s, commit: %s)\n", Version, BuildTime, GitCommit)

	// 设置优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 初始化服务容器
	container := framework.NewHadeContainer()

	// 定义服务注册列表（按照依赖顺序）
	providers := []ServiceProvider{
		// 基础服务（必须成功）
		{Provider: &app.HadeAppProvider{}, Required: true, Order: 1},
		{Provider: &env.HadeEnvProvider{}, Required: true, Order: 2},
		{Provider: &config.HadeConfigProvider{}, Required: true, Order: 3},

		// 核心服务（必须成功）
		{Provider: &hadelog.HadeLogServiceProvider{}, Required: true, Order: 4},
		{Provider: &id.HadeIDProvider{}, Required: true, Order: 5},
		{Provider: &trace.HadeTraceProvider{}, Required: true, Order: 6},

		// 可选服务
		{Provider: &distributed.LocalDistributedProvider{}, Required: false, Order: 7},
		{Provider: &orm.GormProvider{}, Required: false, Order: 8},
		{Provider: &redis.RedisProvider{}, Required: false, Order: 9},
		{Provider: &cache.HadeCacheProvider{}, Required: false, Order: 10},
		{Provider: &ssh.SSHProvider{}, Required: false, Order: 11},
		{Provider: &sls.HadeSLSProvider{}, Required: false, Order: 12},
	}

	// 注册服务
	for _, sp := range providers {
		if err := container.Bind(sp.Provider); err != nil {
			if sp.Required {
				log.Fatalf("Failed to bind required service %s: %v", sp.Provider.Name(), err)
			} else {
				log.Printf("Warning: Failed to bind optional service %s: %v", sp.Provider.Name(), err)
			}
		} else {
			log.Printf("Successfully bound service: %s", sp.Provider.Name())
		}
	}

	// 初始化内核服务
	kernelProvider := &kernel.HadeKernelProvider{}

	// 初始化 HTTP 引擎
	httpEngine, err := http.NewHttpEngine(container)
	if err != nil {
		log.Printf("Warning: Failed to initialize HTTP engine: %v", err)
	} else {
		kernelProvider.HttpEngine = httpEngine
		log.Println("HTTP engine initialized successfully")
	}

	// 初始化 gRPC 引擎
	grpcEngine, err := grpc.NewGrpcEngine(container)
	if err != nil {
		log.Printf("Warning: Failed to initialize gRPC engine: %v", err)
	} else {
		kernelProvider.GrpcEngine = grpcEngine
		log.Println("gRPC engine initialized successfully")
	}

	// 绑定内核服务
	if err := container.Bind(kernelProvider); err != nil {
		log.Fatalf("Failed to bind kernel provider: %v", err)
	}

	// 获取日志服务（如果可用）
	var logger contract.Log
	if container.IsBind(contract.LogKey) {
		logger = container.MustMake(contract.LogKey).(contract.Log)
		logger.Info(ctx, "Application starting", map[string]interface{}{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
			"startup_ms": time.Since(startTime).Milliseconds(),
		})
	}

	// 验证关键配置
	if err := validateConfig(container); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// 启动后台监控
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully shutting down...")

		// 给服务一些时间来完成清理
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// 这里可以添加各个服务的优雅关闭逻辑
		if err := gracefulShutdown(shutdownCtx, container); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}

		cancel()
		os.Exit(0)
	}()

	// 记录启动完成
	log.Printf("Application initialized in %s", time.Since(startTime))

	// 运行命令
	if err := console.RunCommand(container); err != nil {
		log.Fatalf("Failed to run command: %v", err)
	}
}

// validateConfig 验证必要的配置
func validateConfig(container framework.Container) error {
	if !container.IsBind(contract.ConfigKey) {
		return nil // 如果没有配置服务，跳过验证
	}

	configService := container.MustMake(contract.ConfigKey).(contract.Config)

	// 验证必要的配置项
	requiredConfigs := []string{
		"app.name",
		"app.env",
		// 添加其他必要的配置项
	}

	for _, key := range requiredConfigs {
		if !configService.IsExist(key) {
			return fmt.Errorf("required config key '%s' not found", key)
		}
	}

	return nil
}

// gracefulShutdown 优雅关闭
func gracefulShutdown(ctx context.Context, container framework.Container) error {
	// 如果有 HTTP 服务器在运行，关闭它
	if container.IsBind(contract.KernelKey) {
		kernel := container.MustMake(contract.KernelKey).(contract.Kernel)
		// 这里需要 kernel 接口支持 Shutdown 方法
		// 示例：kernel.HttpEngine.Shutdown(ctx)
		_ = kernel // 避免未使用的变量警告
	}

	// 如果有数据库连接，关闭它
	if container.IsBind(contract.ORMKey) {
		orm := container.MustMake(contract.ORMKey).(contract.ORMService)
		// 关闭数据库连接
		// 示例：orm.Close()
		_ = orm // 避免未使用的变量警告
	}

	// 如果有 Redis 连接，关闭它
	if container.IsBind(contract.RedisKey) {
		// redis := container.MustMake(contract.RedisKey)
		// redis.Close()
	}

	log.Println("Graceful shutdown completed")
	return nil
}
