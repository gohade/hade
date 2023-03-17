<template><div><h1 id="安装" tabindex="-1"><a class="header-anchor" href="#安装" aria-hidden="true">#</a> 安装</h1>
<hr>
<h2 id="可执行文件" tabindex="-1"><a class="header-anchor" href="#可执行文件" aria-hidden="true">#</a> 可执行文件</h2>
<p>我们有两种方式来获取可执行的hade文件，第一种是直接下载对应操作系统的hade文件，另外一种是下载源码自己编译</p>
<h3 id="直接下载" tabindex="-1"><a class="header-anchor" href="#直接下载" aria-hidden="true">#</a> 直接下载</h3>
<p>下载地址：
xxx</p>
<p>将生成的可执行文件 hade 放到 $PATH 目录中：
<code v-pre>cp hade /usr/local/bin/</code></p>
<h3 id="源码编译" tabindex="-1"><a class="header-anchor" href="#源码编译" aria-hidden="true">#</a> 源码编译</h3>
<p>下载 git 地址：<code v-pre>git@github.com/jianfengye/hade:cloud/hade.git</code> 到目录 hade</p>
<p>在 hade 目录中运行命令 <code v-pre>go run main.go build self</code></p>
<p>将生成的可执行文件 hade 放到 $PATH 目录中：
<code v-pre>cp hade /usr/local/bin/</code></p>
<h2 id="初始化项目" tabindex="-1"><a class="header-anchor" href="#初始化项目" aria-hidden="true">#</a> 初始化项目</h2>
<p>使用命令 <code v-pre>hade new [app]</code> 在当前目录创建子项目</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace]$ hade new --help
create a new app

Usage:
  hade new [app] [flags]

Aliases:
  new, create, init

Flags:
  -f, --force        if app exist, overwrite app, default: false
  -h, --help         help for new
  -m, --mod string   go mod name, default: folder name
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>这个创建新的hade项目的整个过程是交互式的。你可以根据命令行提示一步步进行。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>➜  hade git:(dev/feature-new) ./hade new
? 请输入目录名称： testdemo
? 请输入模块名称(go.mod中的module, 默认为文件夹名称)：
hade源码从github.com中下载，正在检测到github.com的连接
github.Rate{Limit:60, Remaining:43, Reset:github.Timestamp{2022-12-11 17:00:59 +0800 CST}}
hade源码从github.com中下载，github.com的连接正常
最新的5个版本
v1.0.1
v1.0.0
v0.0.3
v0.0.2
v0.0.1
? 请输入一个版本(更多可以参考 https://github.com/gohade/hade/releases，默认为最新版本)：
====================================================
开始进行创建应用操作
创建目录： /Users/jianfengye/Documents/workspace/gohade/hade/testdemo
应用名称： testdemo
hade框架版本： v1.0.1
创建临时目录 /Users/jianfengye/Documents/workspace/gohade/hade/template-hade-v1.0.1-1670746842
下载zip包到template.zip
解压zip包
删除临时文件夹 /Users/jianfengye/Documents/workspace/gohade/hade/template-hade-v1.0.1-1670746842
删除.git目录
删除framework目录
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/app/http/module/demo/api.go
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/app/http/module/demo/mapper.go
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/app/http/route.go
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/app/http/swagger.go
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/docs/guide/app.md
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/go.mod
更新文件:/Users/jianfengye/Documents/workspace/gohade/hade/testdemo/main.go
创建应用结束
目录： /Users/jianfengye/Documents/workspace/gohade/hade/testdemo
====================================================
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><blockquote>
<p>注意： hade new 依赖github.com的开放平台接口，由于无帐号的开放平台接口有调用次数限制，所以有的时候，会要求你输入github的帐号密码，这是正常的。输出如下：</p>
<div class="language-hade源码从github.com中下载，正在检测到github.com的连接 line-numbers-mode" data-ext="hade源码从github.com中下载，正在检测到github.com的连接"><pre v-pre class="language-hade源码从github.com中下载，正在检测到github.com的连接"><code> github.Rate{Limit:60, Remaining:0, Reset:github.Timestamp{2022-12-11 16:35:25 +0800 CST}}
 错误提示：GET https://api.github.com/repos/gohade/hade/releases?page=1&amp;per_page=10: 403 API rate limit exceeded for 203.205.141.12. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.) [rate reset in 6m14s]
 说明你的出口ip遇到github的调用限制，可以使用github.com帐号登录方式来增加调用次数
 ? 请输入github帐号用户名： jianfengye
 ? 请输入github帐号密码： ********
 github.Rate{Limit:60, Remaining:59, Reset:github.Timestamp{2022-12-11 17:29:19 +0800 CST}}
 hade源码从github.com中下载，github.com的连接正常
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div></blockquote>
<p>创建项目成功后，会在执行目录下创建一个子目录，里面是hade的指定版本的代码。可以直接开始hade之旅。</p>
<h2 id="运行项目" tabindex="-1"><a class="header-anchor" href="#运行项目" aria-hidden="true">#</a> 运行项目</h2>
<p>接下来，可以通过命令 <code v-pre>go run main.go</code> 看到如下信息：</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ go run main.go
hade commands

Usage:
  hade [command]

Available Commands:
  app         start app serve
  build       build hade command
  command     all about commond
  cron        about cron command
  deploy      deploy app by ssh
  dev         dev mode
  env         get current environment
  help        get help info
  middleware  hade middleware
  new         create a new app
  provider    about hade service provider
  swagger     swagger operator

Flags:
  -h, --help   help for hade

Use "hade [command] --help" for more information about a command.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>至此，项目安装成功。</p>
</div></template>


