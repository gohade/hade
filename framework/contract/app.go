package contract

// AppKey 定义字符串凭证
const AppKey = "hade:app"

// App 定义接口
type App interface {
	// AppID 表示当前这个app的唯一id, 可以用于分布式锁等
	AppID() string
	// Version 定义当前版本
	Version() string

	//BaseFolder 定义项目基础地址
	BaseFolder() string
	// ConfigFolder 定义了配置文件的路径
	ConfigFolder() string
	// LogFolder 定义了日志所在路径
	LogFolder() string
	// ProviderFolder 定义业务自己的服务提供者地址
	ProviderFolder() string
	// MiddlewareFolder 定义业务自己定义的中间件
	MiddlewareFolder() string
	// CommandFolder 定义业务定义的命令
	CommandFolder() string
	// RuntimeFolder 定义业务的运行中间态信息
	RuntimeFolder() string
	// TestFolder 存放测试所需要的信息
	TestFolder() string
	// DeployFolder 存放部署的时候创建的文件夹
	DeployFolder() string

	// AppFolder 定义业务代码所在的目录，用于监控文件变更使用
	AppFolder() string
	// LoadAppConfig 加载新的AppConfig，key为对应的函数转为小写下划线，比如ConfigFolder => config_folder
	LoadAppConfig(kv map[string]string)
}
