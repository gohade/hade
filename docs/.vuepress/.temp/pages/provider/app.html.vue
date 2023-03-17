<template><div><h1 id="hade-app" tabindex="-1"><a class="header-anchor" href="#hade-app" aria-hidden="true">#</a> hade:app</h1>
<p>提供基础的 app 框架目录结构</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>package contract

// AppKey is the key in container
const AppKey = &quot;hade:app&quot;

// App define application structure
type App interface {
	// application version
	Version() string
	// base path which is the base folder
	BasePath() string
	// config folder which contains config
	ConfigPath() string
	// environmentPath which contain .env
	EnvironmentPath() string
	// storagePath define storage folder
	StoragePath() string
	// logPath define logPath
	LogPath() string
	// frameworkPath define frameworkPath
	FrameworkPath() string
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div></div></template>


