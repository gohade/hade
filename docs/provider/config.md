# hade:config

提供基础的配置文件获取方法

``` golang
package contract

import "time"

const (
	// ConfigKey is config key in container
	ConfigKey = "hade:config"
)

// Config define setting from files, it support key contains dov。
// for example:
// .Get("user.name")
// suggest use yaml format, https://yaml.org/spec/1.2/spec.html
type Config interface {
	// IsExist check setting is exist
	IsExist(key string) bool

	// Get a new interface
	Get(key string) interface{}
	// GetBool get bool type
	GetBool(key string) bool
	// GetInt get Int type
	GetInt(key string) int
	// GetFloat64 get float64
	GetFloat64(key string) float64
	// GetTime get time type
	GetTime(key string) time.Time
	// GetString get string typen
	GetString(key string) string

	// GetIntSlice get int slice type
	GetIntSlice(key string) []int
	// GetStringSlice get string slice type
	GetStringSlice(key string) []string

	// GetStringMap get map which key is string, value is interface
	GetStringMap(key string) map[string]interface{}
	// GetStringMapString get map which key is string, value is string
	GetStringMapString(key string) map[string]string
	// GetStringMapStringSlice get map which key is string, value is string slice
	GetStringMapStringSlice(key string) map[string][]string

	// Load a config to a struct, val should be an pointer
	Load(key string, val interface{}) error
}

```