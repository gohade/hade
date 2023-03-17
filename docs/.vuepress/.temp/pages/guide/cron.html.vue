<template><div><h1 id="定时任务" tabindex="-1"><a class="header-anchor" href="#定时任务" aria-hidden="true">#</a> 定时任务</h1>
<h2 id="指南" tabindex="-1"><a class="header-anchor" href="#指南" aria-hidden="true">#</a> 指南</h2>
<p>hade 中的定时任务是以命令的形式存在。hade 中也定义了一个命令 <code v-pre>./hade cron</code> 来对定时任务服务进行管理。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>about cron command

Usage:
  hade cron [flags]
  hade cron [command]

Available Commands:
  list        list all cron command
  restart     restart cron command
  start       start cron command
  state       cron serve state
  stop        stop cron command

Flags:
  -h, --help   help for cron

Use "hade cron [command] --help" for more information about a command.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h1 id="创建" tabindex="-1"><a class="header-anchor" href="#创建" aria-hidden="true">#</a> 创建</h1>
<p>创建一个定时任务和创建命令（command）是一致的。具体参考<a href="/guide/command">command</a></p>
<h1 id="挂载" tabindex="-1"><a class="header-anchor" href="#挂载" aria-hidden="true">#</a> 挂载</h1>
<p>和挂载命令稍微有些不同，使用的方法是 <code v-pre>AddCronCommand</code></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>rootCmd.AddCronCommand("* * * * *", command.DemoCommand)
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><h1 id="查询" tabindex="-1"><a class="header-anchor" href="#查询" aria-hidden="true">#</a> 查询</h1>
<p>查询哪些定时任务挂载在服务上，使用命令 <code v-pre>./hade cron list</code></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade cron list
* * * * *  demo  demo
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div></div></div><h1 id="启动" tabindex="-1"><a class="header-anchor" href="#启动" aria-hidden="true">#</a> 启动</h1>
<p>使用命令 <code v-pre>./hade cron start</code> 启动一个定时服务</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade cron start
start cron job
[PID] 35453
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>也可以通过 <code v-pre>./hade cron start -d</code> 使用 deamon 模式启动一个定时服务</p>
<p>定时服务的输出记录在 <code v-pre>/storage/log/cron.log</code></p>
<p>进程 id 记录在 <code v-pre>/storage/pid/app.pid</code></p>
<h1 id="状态" tabindex="-1"><a class="header-anchor" href="#状态" aria-hidden="true">#</a> 状态</h1>
<p>使用 deamon 模式启动定时服务的时候，可以使用命令 <code v-pre>./hade cron state</code> 查询定时任务状态</p>
<h1 id="停止" tabindex="-1"><a class="header-anchor" href="#停止" aria-hidden="true">#</a> 停止</h1>
<p>使用 deamon 模式启动定时服务的时候，可以使用命令 <code v-pre>./hade cron stop</code> 停止定时任务</p>
<h1 id="重启" tabindex="-1"><a class="header-anchor" href="#重启" aria-hidden="true">#</a> 重启</h1>
<p>使用 deamon 模式启动定时服务的时候，可以使用命令 <code v-pre>./hade cron restart</code> 重启定时任务</p>
<div class="custom-container tip"><p class="custom-container-title">TIP</p>
<p>如果程序还未启动，调用 restart 命令，效果和 start 命令一样，deamon 模式启动定时服务</p>
</div>
</div></template>


