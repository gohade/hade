# main.go 优化建议

## 当前代码存在的问题

1. **错误处理被忽略**
   - 所有的 `container.Bind()` 调用都使用 `_ =` 忽略了错误
   - HTTP 和 gRPC 引擎初始化失败时静默跳过
   - `console.RunCommand()` 的错误也被忽略

2. **缺少启动日志**
   - 没有记录应用启动信息
   - 服务注册成功/失败没有日志
   - 缺少版本信息输出

3. **没有优雅退出机制**
   - 没有处理系统信号（SIGINT, SIGTERM）
   - 资源清理逻辑缺失
   - 可能导致数据丢失或连接泄露

4. **服务注册顺序问题**
   - 服务之间可能存在依赖关系
   - 当前是硬编码的顺序，不灵活

5. **缺少配置验证**
   - 没有检查必要的配置项是否存在
   - 可能导致运行时错误

## 优化方案

### 1. 添加错误处理

```go
// 原代码
_ = container.Bind(&app.HadeAppProvider{})

// 优化后
if err := container.Bind(&app.HadeAppProvider{}); err != nil {
    log.Fatalf("Failed to bind App provider: %v", err)
}
```

### 2. 区分必须和可选服务

```go
// 定义服务类型
type ServiceProvider struct {
    Provider framework.ServiceProvider
    Required bool   // 是否必须成功
    Order    int    // 注册顺序
}

// 批量注册
providers := []ServiceProvider{
    // 必须的基础服务
    {Provider: &app.HadeAppProvider{}, Required: true, Order: 1},
    {Provider: &env.HadeEnvProvider{}, Required: true, Order: 2},
    // 可选的服务
    {Provider: &redis.RedisProvider{}, Required: false, Order: 9},
}
```

### 3. 添加优雅退出

```go
// 监听系统信号
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigChan
    log.Println("Shutting down gracefully...")
    
    // 关闭 HTTP 服务器
    // 关闭数据库连接
    // 清理其他资源
    
    os.Exit(0)
}()
```

### 4. 添加启动日志和版本信息

```go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    startTime := time.Now()
    fmt.Printf("Hade Framework %s (built: %s, commit: %s)\n", 
        Version, BuildTime, GitCommit)
    
    // ... 其他代码 ...
    
    log.Printf("Application started in %s", time.Since(startTime))
}
```

### 5. 添加配置验证

```go
func validateConfig(container framework.Container) error {
    config := container.MustMake(contract.ConfigKey).(contract.Config)
    
    requiredKeys := []string{
        "app.name",
        "app.env",
        "app.port",
    }
    
    for _, key := range requiredKeys {
        if !config.IsExist(key) {
            return fmt.Errorf("required config '%s' not found", key)
        }
    }
    return nil
}
```

### 6. 使用服务注册表模式

```go
// 可以创建一个服务注册表
type ServiceRegistry struct {
    providers []ServiceProvider
}

func NewServiceRegistry() *ServiceRegistry {
    return &ServiceRegistry{
        providers: []ServiceProvider{
            // 在这里定义所有服务
        },
    }
}

func (r *ServiceRegistry) RegisterAll(container framework.Container) error {
    for _, sp := range r.providers {
        if err := container.Bind(sp.Provider); err != nil {
            if sp.Required {
                return fmt.Errorf("bind %s: %w", sp.Provider.Name(), err)
            }
            log.Printf("Warning: %v", err)
        }
    }
    return nil
}
```

### 7. 添加健康检查

```go
func healthCheck(container framework.Container) error {
    // 检查数据库连接
    if container.IsBind(contract.ORMKey) {
        orm := container.MustMake(contract.ORMKey).(contract.ORMService)
        // 执行简单查询测试连接
    }
    
    // 检查 Redis 连接
    if container.IsBind(contract.RedisKey) {
        // 测试 Redis 连接
    }
    
    return nil
}
```

### 8. 环境特定的初始化

```go
env := container.MustMake(contract.EnvKey).(contract.Env)

switch env.AppEnv() {
case "production":
    // 生产环境特定配置
    // 例如：启用 SLS 日志服务
case "development":
    // 开发环境特定配置
    // 例如：启用调试模式
}
```

## 完整的优化示例

查看 `main_optimized.go` 文件，其中包含了所有这些优化的完整实现。

## 构建时注入版本信息

在构建时可以使用 ldflags 注入版本信息：

```bash
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date +%Y%m%d%H%M%S) -X main.GitCommit=$(git rev-parse HEAD)"
```

## 性能监控建议

1. 添加启动时间统计
2. 记录每个服务的初始化时间
3. 使用 pprof 进行性能分析
4. 添加 metrics 收集

## 安全性建议

1. 避免在日志中打印敏感配置
2. 使用环境变量管理敏感信息
3. 添加配置加密支持

## 测试建议

1. 为 main 函数添加集成测试
2. 模拟服务注册失败的场景
3. 测试优雅退出逻辑
4. 测试配置验证逻辑