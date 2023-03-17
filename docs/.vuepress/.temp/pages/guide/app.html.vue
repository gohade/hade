<template><div><h1 id="运行" tabindex="-1"><a class="header-anchor" href="#运行" aria-hidden="true">#</a> 运行</h1>
<h2 id="命令" tabindex="-1"><a class="header-anchor" href="#命令" aria-hidden="true">#</a> 命令</h2>
<p>这里的运行是运行整个 app，这个 app 可以只包含后端，也可以只包含前端，但是后端也是隐藏在前端后面运行。具体可以参考
app/http/route.go</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>package http

import (
	"github.com/gohade/hade/app/http/controller/demo"
	"github.com/gohade/hade/framework/gin"
)

func Routes(r *gin.Engine) {
	r.Static("/dist/", "./dist/")
	r.GET("/demo/demo", demo.Demo)
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>运行相关的命令为 app。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade app
start app serve

Usage:
  hade app [flags]
  hade app [command]

Available Commands:
  restart     restart app server
  start       start app server
  state       get app pid
  stop        stop app server

Flags:
  -h, --help   help for app

Use "hade app [command] --help" for more information about a command.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="启动" tabindex="-1"><a class="header-anchor" href="#启动" aria-hidden="true">#</a> 启动</h2>
<p>可以使用 <code v-pre>./hade app start</code> 启动一个应用。</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>成功启动进程: hade app
进程pid: 39327
监听地址: http://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>也可以使用 <code v-pre>./hade app start -d</code> 使用 deamon 模式启动一个应用。应用名称为 <code v-pre>hade app</code></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade app start -d
成功启动进程: hade app
进程pid: 41021
监听地址: http://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>app 应用的输出记录在 <code v-pre>/storage/log/app.log</code></p>
<p>进程 id 记录在 <code v-pre>/storage/pid/app.pid</code></p>
<h2 id="状态" tabindex="-1"><a class="header-anchor" href="#状态" aria-hidden="true">#</a> 状态</h2>
<p>当使用 deamon 模式启动的时候，需要查看当前应用是否有启动，如果启动了，进程号是多少，可以使用命令 <code v-pre>./hade app state</code></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade app state
app服务已经启动, pid: 41021
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="重启" tabindex="-1"><a class="header-anchor" href="#重启" aria-hidden="true">#</a> 重启</h2>
<p>当使用 deamon 模式启动的时候，需要重启应用，可以使用命令 <code v-pre>./hade app restart</code></p>
<div class="custom-container tip"><p class="custom-container-title">TIP</p>
<p>如果程序还未启动，调用 restart 命令，效果和 start 命令一样，deamon 模式启动应用</p>
</div>
<h2 id="停止" tabindex="-1"><a class="header-anchor" href="#停止" aria-hidden="true">#</a> 停止</h2>
<p>当使用 deamon 模式启动的时候，需要关闭应用，可以使用命令 <code v-pre>./hade app stop</code></p>
<h2 id="进程运行基础配置" tabindex="-1"><a class="header-anchor" href="#进程运行基础配置" aria-hidden="true">#</a> 进程运行基础配置</h2>
<p>在启动进程的时候，我们会需要定义一些配置项，这些配置项决定进程的运行环境（比如日志存放位置，运行态信息存放位置，配置文件存放位置等）。</p>
<p>这里我们提供了3种配置方式来设置这些基础配置，包括环境变量设置，命令行参数设置，配置文件设置。</p>
<p>这三种配置方式的优先级为：命令行参数 &gt; 环境变量 &gt; 配置文件</p>
<p>具体的配置项常用的如下，具体更多可以参考 framework/provider/app/service.go ：</p>
<ul>
<li>运行中间信息存放目录
<ul>
<li>命令行参数：--runtime_folder</li>
<li>环境变量：RUNTIME_FOLDER</li>
<li>配置文件：app.path.runtime_folder</li>
<li>不设置默认为：运行信息基础目录 + runtime</li>
</ul>
</li>
<li>日志存放目录
<ul>
<li>命令行参数：--log_folder</li>
<li>环境变量：LOG_FOLDER</li>
<li>配置文件：app.path.log_folder</li>
<li>不设置默认为：运行信息基础目录 + log</li>
</ul>
</li>
<li>运行信息基础目录
<ul>
<li>命令行参数：--storage_folder</li>
<li>环境变量：STORAGE_FOLDER</li>
<li>配置文件：app.path.storage_folder</li>
<li>不设置默认为：基础目录 + storage</li>
</ul>
</li>
<li>配置文件地址
<ul>
<li>命令行参数：--config_folder</li>
<li>环境变量：CONFIG_FOLDER</li>
<li>配置文件：app.path.config_folder</li>
<li>不设置默认为：基础目录 + config</li>
</ul>
</li>
<li>基础目录
<ul>
<li>命令行参数：--base_folder</li>
<li>环境变量：BASE_FOLDER</li>
<li>配置文件：app.path.base_folder</li>
<li>不设置默认为：当前执行目录</li>
</ul>
</li>
</ul>
<h3 id="环境变量设置" tabindex="-1"><a class="header-anchor" href="#环境变量设置" aria-hidden="true">#</a> 环境变量设置</h3>
<p>在启动进程的时候进行环境变量的设置。比如</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>STORAGE_FOLDER=/Users/jianfengye/Documents/workspace/gohade/hade/teststorage ./hade app start
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><h3 id="命令行参数设置" tabindex="-1"><a class="header-anchor" href="#命令行参数设置" aria-hidden="true">#</a> 命令行参数设置</h3>
<p>在命令行参数中设置。比如</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>./hade app start --storage_folder=/Users/jianfengye/Documents/workspace/gohade/hade/teststorage/ -d
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><h3 id="配置文件设置" tabindex="-1"><a class="header-anchor" href="#配置文件设置" aria-hidden="true">#</a> 配置文件设置</h3>
<p>在配置文件config/${env}/app.yaml中配置：</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>path:
storage_folder: "/Users/jianfengye/Documents/workspace/gohade/hade/teststorage/"
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div></div></div></div></template>


