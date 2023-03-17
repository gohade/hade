<template><div><h1 id="调试模式" tabindex="-1"><a class="header-anchor" href="#调试模式" aria-hidden="true">#</a> 调试模式</h1>
<h2 id="命令" tabindex="-1"><a class="header-anchor" href="#命令" aria-hidden="true">#</a> 命令</h2>
<p>hade 框架自带调试模式，不管是前端还是后端，都可以启动调试模式，边修改代码，边编译运行服务。</p>
<p>对应的命令为 <code v-pre>./hade dev</code></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev
dev mode

Usage:
  hade dev [flags]
  hade dev [command]

Available Commands:
  all         dev mode from both frontend and backend
  backend     dev mode for backend, hot reload
  frontend    dev mode for frontend

Flags:
  -h, --help   help for dev

Use "hade dev [command] --help" for more information about a command.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><ul>
<li>调试前端</li>
<li>调试后端</li>
<li>同时调试</li>
</ul>
<h2 id="调试前端" tabindex="-1"><a class="header-anchor" href="#调试前端" aria-hidden="true">#</a> 调试前端</h2>
<p>使用命令 <code v-pre>./hade dev frontend</code></p>
<p>要求当前编译机器安装 npm 软件，并且当前项目已经运行了 npm install，安装完成前端依赖。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev frontend

> hade@0.1.0 serve /Users/Documents/workspace/hade_workspace/demo5
> vue-cli-service serve

 INFO  Starting development server...
98% after emitting

 DONE  Compiled successfully in 2589ms                                                                                                     下午6:07:06


  App running at:
  - Local:   http://localhost:8080
  - Network: http://172.24.34.34:8080

  Note that the development build is not optimized.
  To create a production build, run npm run build.
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>实际上是调用 <code v-pre>npm run dev</code> 来调试前端</p>
<h2 id="调试后端" tabindex="-1"><a class="header-anchor" href="#调试后端" aria-hidden="true">#</a> 调试后端</h2>
<p>使用命令 <code v-pre>./hade dev backend</code></p>
<p>要求当前编译机器安装 go 软件，版本 &gt; 1.3。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$  ./hade dev backend
./hade dev backend
build success please run ./hade direct
backend server: http://127.0.0.1:15060
proxy backend server: http://0.0.0.0:8073
[PID] 29034
app serve url: http://127.0.0.1:15060
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>可以通过 proxy backend server 地址进行访问：</p>
<p><code v-pre>http://0.0.0.0:8073/demo/demo</code></p>
<div class="custom-container tip"><p class="custom-container-title">TIP</p>
<p>后端调试默认是最后一次操作后3秒启动后端编译启动命令。</p>
<p>hade 也允许通过配置修改这个等待时间。</p>
<p>可以配置 <code v-pre>development/app.yaml</code> 里面的 <code v-pre>dev_fresh</code> 参数修改这个等待时间。</p>
</div>
<h1 id="同时调试" tabindex="-1"><a class="header-anchor" href="#同时调试" aria-hidden="true">#</a> 同时调试</h1>
<p>也可以选择同时调试，这个时候会同时运行调试前端和调试后端的程序</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev all

> hade@0.1.0 serve /Users/Documents/workspace/hade_workspace/demo5
> vue-cli-service serve

 INFO  Starting development server...
build success please run ./hade direct
backend server: http://127.0.0.1:19866
proxy backend server: http://0.0.0.0:8073
proxy frontend server: http://0.0.0.0:8073/dist/#/
[PID] 29761
app serve url: http://127.0.0.1:19866
98% after emitting

 DONE  Compiled successfully in 1421ms                                                                                                     下午6:19:51


  App running at:
  - Local:   http://localhost:19073
  - Network: http://172.24.34.34:19073

  Note that the development build is not optimized.
  To create a production build, run npm run build.

[GIN] 2020/09/16 - 18:20:26 | 200 |     134.079µs |       127.0.0.1 | GET      /demo/demo

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>前端和后端的访问地址分别为：</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>proxy backend server: http://0.0.0.0:8073
proxy frontend server: http://0.0.0.0:8073/dist/#/
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div></div></div></div></template>


