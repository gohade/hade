<template><div><h1 id="中间件" tabindex="-1"><a class="header-anchor" href="#中间件" aria-hidden="true">#</a> 中间件</h1>
<h2 id="指南" tabindex="-1"><a class="header-anchor" href="#指南" aria-hidden="true">#</a> 指南</h2>
<p>hade 的 HTTP 路由服务并没有自己开发，而是使用 gin。gin 生态已经有非常完善的(中间件体系)[https://github.com/gin-contrib]。</p>
<p>我们没有必要重新开发这些中间件。所以 hade 框架提供了 middleware 命令来获取这些中间件。（不能直接使用go get 的方式来获取，因为 hade 在 gin 基础上做了一些微调）</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade middleware
hade middleware

Usage:
  hade middleware [flags]
  hade middleware [command]

Available Commands:
  add         add middleware to app, https://github.com/gin-contrib/[middleware].git
  list        list all installed middleware
  remove      remove middleware from app

Flags:
  -h, --help   help for middleware

Use "hade middleware [command] --help" for more information about a command.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="安装" tabindex="-1"><a class="header-anchor" href="#安装" aria-hidden="true">#</a> 安装</h2>
<p>可以安装 https://github.com/gin-contrib/ 项目中的任何中间件，使用命令 <code v-pre>./hade middleware add gzip</code></p>
<p>命令会从 https://github.com/gin-contrib/gzip.git 项目中下载中间件，并且安装到 <code v-pre>app/http/middleware</code> 中。</p>
<h2 id="查询" tabindex="-1"><a class="header-anchor" href="#查询" aria-hidden="true">#</a> 查询</h2>
<p>检查目前已经安装了哪些中间件，可以使用命令 <code v-pre>./hade middleware list</code></p>
<h2 id="删除" tabindex="-1"><a class="header-anchor" href="#删除" aria-hidden="true">#</a> 删除</h2>
<p>删除某个中间件，可以使用命令 <code v-pre>./hade middleware remove</code></p>
</div></template>


