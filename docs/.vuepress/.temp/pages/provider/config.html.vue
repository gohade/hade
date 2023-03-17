<template><div><h1 id="hade-config" tabindex="-1"><a class="header-anchor" href="#hade-config" aria-hidden="true">#</a> hade:config</h1>
<p>提供基础的配置文件获取方法</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>package contract

import &quot;time&quot;

const (
	// ConfigKey is config key in container
	ConfigKey = &quot;hade:config&quot;
)

// Config define setting from files, it support key contains dov。
// for example:
// .Get(&quot;user.name&quot;)
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

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div></div></template>


