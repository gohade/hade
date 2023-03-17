<template><div><h1 id="hade-env" tabindex="-1"><a class="header-anchor" href="#hade-env" aria-hidden="true">#</a> hade:env</h1>
<p>提供环境变量相关方法</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>package contract

const (
	// EnvProduction represent the environment which build for production
	EnvProduction = &quot;production&quot;
	// EnvTesting represent the environment which build for test
	EnvTesting = &quot;testing&quot;
	// EnvDevelopment represent the environment which build for development
	EnvDevelopment = &quot;development&quot;

	// EnvKey is the key in container
	EnvKey = &quot;hade:env&quot;
)

// Env define golang run enviornment
// it set some config which want ignored in git
type Env interface {
	// AppEnv get current environment
	AppEnv() string

	// IsExist check setting is exist
	IsExist(string) bool
	// Get environment setting, if not exist, return &quot;&quot;
	Get(string) string
	// All return all settings
	All() map[string]string
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div></div></template>


