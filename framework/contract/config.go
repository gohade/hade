package contract

import "time"

const (
	// ConfigKey is config key in container
	ConfigKey = "hade:config"
)

// Config 定义了配置文件服务，读取配置文件，支持点分割的路径读取
// 例如: .Get("app.name") 表示从app文件中读取name属性
// 建议使用 yaml 属性, https://yaml.org/spec/1.2/spec.html
type Config interface {
	// IsExist 检查一个属性是否存在
	IsExist(key string) bool

	// Get 获取一个属性值
	Get(key string) interface{}
	// GetBool 获取一个bool属性
	GetBool(key string) bool
	// GetInt 获取一个int属性
	GetInt(key string) int
	// GetFloat64 获取一个float64属性
	GetFloat64(key string) float64
	// GetTime 获取一个time属性
	GetTime(key string) time.Time
	// GetString 获取一个string属性
	GetString(key string) string
	// GetIntSlice 获取一个int数组属性
	GetIntSlice(key string) []int
	// GetStringSlice 获取一个string数组
	GetStringSlice(key string) []string
	// GetStringMap 获取一个string为key，interface为val的map
	GetStringMap(key string) map[string]interface{}
	// GetStringMapString 获取一个string为key，string为val的map
	GetStringMapString(key string) map[string]string
	// GetStringMapStringSlice 获取一个string为key，数组string为val的map
	GetStringMapStringSlice(key string) map[string][]string

	// Load 加载配置到某个对象
	Load(key string, val interface{}) error
}
