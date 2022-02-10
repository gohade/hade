package contract

const (
	// EnvProduction 代表生产环境
	EnvProduction = "production"
	// EnvTesting 代表测试环境
	EnvTesting = "testing"
	// EnvDevelopment 代表开发环境
	EnvDevelopment = "development"

	// EnvKey 是环境变量服务字符串凭证
	EnvKey = "hade:env"
)

// Env 定义环境变量的获取服务
type Env interface {
	// AppEnv 获取当前的环境，建议分为development/testing/production
	AppEnv() string

	// IsExist 判断一个环境变量是否有被设置
	IsExist(string) bool
	// Get 获取某个环境变量，如果没有设置，返回""
	Get(string) string
	// All 获取所有的环境变量，.env和运行环境变量融合后结果
	All() map[string]string
}
