# Hade 框架详细数据流和调用关系图

## 1. 应用启动流程

```mermaid
graph TD
    A[main.go] --> B[创建容器<br/>NewHadeContainer]
    B --> C[绑定核心服务提供者]
    C --> D[App Provider]
    C --> E[Env Provider]
    C --> F[Config Provider]
    C --> G[其他 Providers...]
    
    D --> H[初始化 HTTP Engine<br/>基于 Gin]
    D --> I[初始化 gRPC Engine]
    
    H --> J[绑定 Kernel Provider]
    I --> J
    
    J --> K[运行 Console Command<br/>基于 Cobra]
    
    K --> L{命令类型}
    L -->|app start| M[启动 HTTP/gRPC 服务]
    L -->|dev| N[启动开发模式]
    L -->|cron start| O[启动定时任务]
    L -->|其他命令| P[执行相应命令]
```

## 2. 服务注册与依赖注入流程

```mermaid
sequenceDiagram
    participant App as 应用程序
    participant Container as 服务容器
    participant Provider as 服务提供者
    participant Instance as 服务实例
    
    App->>Container: Bind(ServiceProvider)
    Container->>Provider: 检查 IsDefer()
    
    alt 非延迟加载
        Container->>Provider: Boot(container)
        Provider-->>Container: 返回启动状态
        Container->>Provider: Params(container)
        Provider-->>Container: 返回参数
        Container->>Provider: Register(container)
        Provider-->>Container: 返回 NewInstance 函数
        Container->>Instance: 调用 NewInstance(params...)
        Instance-->>Container: 返回实例
        Container->>Container: 存储实例到 instances map
    else 延迟加载
        Container->>Container: 仅存储 Provider 到 providers map
    end
    
    App->>Container: Make(key)
    Container->>Container: 检查 instances map
    
    alt 实例已存在
        Container-->>App: 返回缓存的实例
    else 实例不存在
        Container->>Provider: Boot(container)
        Container->>Provider: Register(container)
        Container->>Instance: 创建新实例
        Container->>Container: 缓存实例
        Container-->>App: 返回新实例
    end
```

## 3. HTTP 请求处理流程

```mermaid
graph LR
    A[客户端请求] --> B[Gin Engine]
    B --> C[全局中间件]
    C --> D[路由匹配]
    D --> E[路由中间件]
    E --> F[Controller]
    
    F --> G{需要服务?}
    G -->|是| H[Container.Make]
    H --> I[获取服务实例]
    I --> J[调用服务方法]
    J --> K[返回结果]
    
    G -->|否| K
    K --> L[Response]
    L --> M[客户端]
```

## 4. 命令执行流程

```mermaid
graph TD
    A[用户输入命令] --> B[Cobra 解析命令]
    B --> C[查找对应 Command]
    C --> D[执行 Command.Run]
    
    D --> E{命令类型}
    
    E -->|app| F[应用管理命令]
    F --> F1[start: 启动服务]
    F --> F2[restart: 重启服务]
    F --> F3[stop: 停止服务]
    F --> F4[state: 查看状态]
    
    E -->|dev| G[开发模式]
    G --> G1[监控文件变化]
    G --> G2[自动重启后端]
    G --> G3[代理前端服务]
    
    E -->|cron| H[定时任务]
    H --> H1[start: 启动定时服务]
    H --> H2[restart: 重启定时服务]
    H --> H3[stop: 停止定时服务]
    H --> H4[state: 查看任务状态]
    H --> H5[list: 列出所有任务]
    
    E -->|provider| I[服务提供者管理]
    I --> I1[list: 列出所有提供者]
    I --> I2[new: 创建新提供者]
    
    E -->|其他| J[其他命令...]
```

## 5. 服务层级依赖关系

```mermaid
graph BT
    subgraph 基础服务层
        A1[App Service]
        A2[Env Service]
        A3[Config Service]
    end
    
    subgraph 核心服务层
        B1[Log Service]
        B2[ID Service]
        B3[Trace Service]
    end
    
    subgraph 数据服务层
        C1[ORM Service]
        C2[Redis Service]
        C3[Cache Service]
    end
    
    subgraph 高级服务层
        D1[Distributed Service]
        D2[SSH Service]
        D3[SLS Service]
    end
    
    subgraph 应用服务层
        E1[HTTP Kernel]
        E2[gRPC Kernel]
        E3[Cron Service]
    end
    
    B1 --> A1
    B1 --> A2
    B1 --> A3
    
    B2 --> A1
    B3 --> A1
    B3 --> B2
    
    C1 --> A3
    C1 --> B1
    
    C2 --> A3
    C2 --> B1
    
    C3 --> C2
    C3 --> B1
    
    D1 --> A1
    D2 --> A3
    D3 --> B1
    
    E1 --> B1
    E1 --> B3
    E2 --> B1
    E2 --> B3
    E3 --> B1
```

## 6. 配置加载流程

```mermaid
sequenceDiagram
    participant App as 应用
    participant ConfigProvider as 配置提供者
    participant EnvProvider as 环境提供者
    participant ConfigFiles as 配置文件
    
    App->>ConfigProvider: Boot()
    ConfigProvider->>EnvProvider: 获取环境变量
    EnvProvider-->>ConfigProvider: 返回环境信息
    
    ConfigProvider->>ConfigProvider: 确定配置目录
    ConfigProvider->>ConfigFiles: 加载配置文件
    
    Note over ConfigFiles: 加载顺序:<br/>1. config.yaml<br/>2. config.{env}.yaml<br/>3. 环境变量覆盖
    
    ConfigFiles-->>ConfigProvider: 返回配置数据
    ConfigProvider->>ConfigProvider: 合并配置
    ConfigProvider-->>App: 配置就绪
    
    App->>ConfigProvider: Get(key)
    ConfigProvider-->>App: 返回配置值
```

## 7. 中间件执行链

```mermaid
graph TD
    A[HTTP 请求] --> B[Recovery 中间件<br/>异常恢复]
    B --> C[Trace 中间件<br/>链路追踪]
    C --> D[Log 中间件<br/>请求日志]
    D --> E[Timeout 中间件<br/>超时控制]
    E --> F[Cost 中间件<br/>耗时统计]
    F --> G[自定义中间件...]
    G --> H[路由处理器]
    
    H --> I[业务逻辑]
    I --> J[响应]
    
    J --> K[中间件后置处理]
    K --> L[返回客户端]
```

## 8. 定时任务执行流程

```mermaid
stateDiagram-v2
    [*] --> 初始化: cron start
    初始化 --> 加载任务: 读取任务配置
    加载任务 --> 验证任务: 检查任务合法性
    
    验证任务 --> 注册任务: 任务有效
    验证任务 --> 错误处理: 任务无效
    
    注册任务 --> 等待执行: 按照 cron 表达式
    等待执行 --> 执行任务: 时间到达
    
    执行任务 --> 记录日志: 执行中
    记录日志 --> 等待执行: 执行完成
    
    等待执行 --> 停止服务: cron stop
    停止服务 --> [*]
    
    错误处理 --> [*]
```

## 9. 开发模式工作流

```mermaid
graph TD
    A[hade dev] --> B[启动开发服务器]
    B --> C[启动文件监控]
    B --> D[启动代理服务器]
    
    C --> E{文件变化?}
    E -->|后端文件| F[重新编译]
    F --> G[重启后端服务]
    
    E -->|前端文件| H[触发前端构建]
    H --> I[热更新浏览器]
    
    D --> J[代理后端 API]
    D --> K[代理前端资源]
    
    G --> L[更新服务]
    I --> L
    L --> M[开发者看到更新]
```

## 10. 服务生命周期

```mermaid
stateDiagram-v2
    [*] --> 未注册: 初始状态
    未注册 --> 已注册: Container.Bind()
    
    已注册 --> 已启动: Provider.Boot()
    已启动 --> 已实例化: Provider.Register()
    
    已实例化 --> 使用中: Container.Make()
    使用中 --> 使用中: 多次调用
    
    state 已注册 {
        [*] --> 延迟加载: IsDefer() = true
        [*] --> 立即加载: IsDefer() = false
        
        延迟加载 --> 等待调用: 存储 Provider
        立即加载 --> 执行实例化: 立即执行
        
        等待调用 --> 执行实例化: 首次 Make()
    }
    
    使用中 --> 销毁: 应用关闭
    销毁 --> [*]
```

这些详细的流程图展示了 Hade 框架的：
- 应用启动和初始化过程
- 依赖注入的完整流程
- HTTP 请求的处理链路
- 命令行工具的执行逻辑
- 服务之间的依赖关系
- 配置文件的加载机制
- 中间件的执行顺序
- 定时任务的生命周期
- 开发模式的工作原理
- 服务的完整生命周期