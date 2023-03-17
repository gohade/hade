<template><div><h1 id="自动部署" tabindex="-1"><a class="header-anchor" href="#自动部署" aria-hidden="true">#</a> 自动部署</h1>
<p>部署自动化其实不是一个框架的刚需，有很多方式可以将一个服务进行自动化部署，比如现在比较流行的 Docker 化或者 CI/CD
流程。但是一些比较个人比较小的项目，比如一个博客、一个官网网站，这些部署流程往往都太庞大了，更需要一个服务，能快速将在开发机器上写好、调试好的程序上传到目标服务器，并且更新应用程序。这就是hade框架实现的发布自动化。</p>
<h2 id="ssh" tabindex="-1"><a class="header-anchor" href="#ssh" aria-hidden="true">#</a> SSH</h2>
<p>所有的部署自动化工具，基本都依赖本地与远端服务器的连接，这个连接可以是 FTP，可以是 HTTP，但是更经常的连接是 SSH 连接。
基本上，SSH 账号是我们拿到 Web 服务器的首要凭证，所以要设计的自动化发布系统也是依赖 SSH 的。</p>
<p>应的配置文件如下 config/testing/ssh.yaml，你可以看看每个配置的说明：</p>
<div class="language-yaml line-numbers-mode" data-ext="yml"><pre v-pre class="language-yaml"><code>
<span class="token key atrule">timeout</span><span class="token punctuation">:</span> 1s
<span class="token key atrule">network</span><span class="token punctuation">:</span> tcp
<span class="token key atrule">web-01</span><span class="token punctuation">:</span>
    <span class="token key atrule">host</span><span class="token punctuation">:</span> 118.190.3.55 <span class="token comment"># ip地址</span>
    <span class="token key atrule">port</span><span class="token punctuation">:</span> <span class="token number">22</span> <span class="token comment"># 端口</span>
    <span class="token key atrule">username</span><span class="token punctuation">:</span> yejianfeng <span class="token comment"># 用户名</span>
    <span class="token key atrule">password</span><span class="token punctuation">:</span> <span class="token string">"123456"</span> <span class="token comment"># 密码</span>
<span class="token key atrule">web-02</span><span class="token punctuation">:</span>
    <span class="token key atrule">network</span><span class="token punctuation">:</span> tcp
    <span class="token key atrule">host</span><span class="token punctuation">:</span> localhost <span class="token comment"># ip地址</span>
    <span class="token key atrule">port</span><span class="token punctuation">:</span> <span class="token number">3306</span> <span class="token comment"># 端口</span>
    <span class="token key atrule">username</span><span class="token punctuation">:</span> jianfengye <span class="token comment"># 用户名</span>
    <span class="token key atrule">rsa_key</span><span class="token punctuation">:</span> <span class="token string">"/Users/user/.ssh/id_rsa"</span>
    <span class="token key atrule">known_hosts</span><span class="token punctuation">:</span> <span class="token string">"/Users/user/.ssh/known_hosts"</span>
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>SSH 的连接方式有两种，一种是直接使用用户名密码来连接远程服务器，还有一种是使用 rsa key 文件来连接远端服务器，所以这里的配置需要同时支持两种配置。对于使用
rsa key 文件的方式，需要设置 rsk_key 的私钥地址和负责安全验证的 known_hosts。</p>
<h2 id="deploy" tabindex="-1"><a class="header-anchor" href="#deploy" aria-hidden="true">#</a> deploy</h2>
<p>我们的 hade 框架是同时支持前后端的开发框架，所以自动化部署是需要同时支持前后端部署的，也就是说它的命令也需要支持前后端的部署，这里我们设计一个显示帮助信息的一级命令./hade
deploy 和四个二级命令：</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>./hade deploy frontend ，部署前端
./hade deploy backend ，部署后端
./hade deploy all ，同时部署前后端
./hade deploy rollback ，部署回滚
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>完整的配置文件在 config/development/deploy.yaml 中：</p>
<div class="language-yaml line-numbers-mode" data-ext="yml"><pre v-pre class="language-yaml"><code>
<span class="token key atrule">connections</span><span class="token punctuation">:</span> <span class="token comment"># 要自动化部署的连接</span>
    <span class="token punctuation">-</span> ssh.web<span class="token punctuation">-</span><span class="token number">01</span>

<span class="token key atrule">remote_folder</span><span class="token punctuation">:</span> <span class="token string">"/home/yejianfeng/coredemo/"</span>  <span class="token comment"># 远端的部署文件夹</span>

<span class="token key atrule">frontend</span><span class="token punctuation">:</span> <span class="token comment"># 前端部署配置</span>
    <span class="token key atrule">pre_action</span><span class="token punctuation">:</span> <span class="token comment"># 部署前置命令</span>
        <span class="token punctuation">-</span> <span class="token string">"pwd"</span>
    <span class="token key atrule">post_action</span><span class="token punctuation">:</span> <span class="token comment"># 部署后置命令</span>
        <span class="token punctuation">-</span> <span class="token string">"pwd"</span>

<span class="token key atrule">backend</span><span class="token punctuation">:</span> <span class="token comment"># 后端部署配置</span>
    <span class="token key atrule">goos</span><span class="token punctuation">:</span> linux <span class="token comment"># 部署目标操作系统</span>
    <span class="token key atrule">goarch</span><span class="token punctuation">:</span> amd64 <span class="token comment"># 部署目标cpu架构</span>
    <span class="token key atrule">pre_action</span><span class="token punctuation">:</span> <span class="token comment"># 部署前置命令</span>
        <span class="token punctuation">-</span> <span class="token string">"pwd"</span>
    <span class="token key atrule">post_action</span><span class="token punctuation">:</span> <span class="token comment"># 部署后置命令</span>
        <span class="token punctuation">-</span> <span class="token string">"chmod 777 /home/yejianfeng/coredemo/hade"</span>
        <span class="token punctuation">-</span> <span class="token string">"/home/yejianfeng/coredemo/hade app restart"</span>
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h3 id="部署前端" tabindex="-1"><a class="header-anchor" href="#部署前端" aria-hidden="true">#</a> 部署前端</h3>
<p>你可以通过命令</p>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code>./hade deploy frontend
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><p>或者 跳过编译环节</p>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code>./hade deploy frontend <span class="token parameter variable">-s</span><span class="token operator">=</span>true
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><p>第一个方法会直接运行npm run build，把前端代码生成在dist目录下，然后把dist目录下的文件上传到远端服务器，然后执行前置命令和后置命令。
而第二个方法会掉过编译，直接把dist目录下的文件上传到远端服务器，然后执行前置命令和后置命令。</p>
<p>下面就是我们的hade官网的部署过程。
我每次都在本地通过vuepress编译docs目录下的markdown到dist目录下，然后通过hade部署到远端服务器。</p>
<div class="language-markdown line-numbers-mode" data-ext="md"><pre v-pre class="language-markdown"><code>➜  hade git:(main) ✗ ./hade deploy frontend -s=true
[Info]	2022-12-14T20:37:46+08:00	"execute pre action start"	map[cmd:pwd connection:ssh.web-01]
[Info]	2022-12-14T20:37:46+08:00	"execute pre action"	map[cmd:pwd connection:ssh.web-01 out:/home/yejianfeng]
[Info]	2022-12-14T20:37:46+08:00	"mkdir: /webroot/hade_doc/dist"	map[]
[Info]	2022-12-14T20:37:46+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/404.html to remote file: /webroot/hade_doc/dist/404.html finish"	map[]
[Info]	2022-12-14T20:37:46+08:00	"mkdir: /webroot/hade_doc/dist/assets"	map[]
[Info]	2022-12-14T20:37:46+08:00	"mkdir: /webroot/hade_doc/dist/assets/css"	map[]
[Info]	2022-12-14T20:37:46+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/css/0.styles.fb3ee4f4.css to remote file: /webroot/hade_doc/dist/assets/css/0.styles.fb3ee4f4.css finish"	map[]
[Info]	2022-12-14T20:37:46+08:00	"mkdir: /webroot/hade_doc/dist/assets/img"	map[]
[Info]	2022-12-14T20:37:46+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/img/search.83621669.svg to remote file: /webroot/hade_doc/dist/assets/img/search.83621669.svg finish"	map[]
[Info]	2022-12-14T20:37:46+08:00	"mkdir: /webroot/hade_doc/dist/assets/js"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/10.95277eaa.js to remote file: /webroot/hade_doc/dist/assets/js/10.95277eaa.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/11.ce56aea9.js to remote file: /webroot/hade_doc/dist/assets/js/11.ce56aea9.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/12.1ab2b8bd.js to remote file: /webroot/hade_doc/dist/assets/js/12.1ab2b8bd.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/13.2165b0a4.js to remote file: /webroot/hade_doc/dist/assets/js/13.2165b0a4.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/14.62f5324b.js to remote file: /webroot/hade_doc/dist/assets/js/14.62f5324b.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/15.af7aad9d.js to remote file: /webroot/hade_doc/dist/assets/js/15.af7aad9d.js finish"	map[]
[Info]	2022-12-14T20:37:47+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/16.e3cf5518.js to remote file: /webroot/hade_doc/dist/assets/js/16.e3cf5518.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/17.2fe064fe.js to remote file: /webroot/hade_doc/dist/assets/js/17.2fe064fe.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/18.0f1b26f9.js to remote file: /webroot/hade_doc/dist/assets/js/18.0f1b26f9.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/19.90ea3d02.js to remote file: /webroot/hade_doc/dist/assets/js/19.90ea3d02.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/2.be84f03d.js to remote file: /webroot/hade_doc/dist/assets/js/2.be84f03d.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/20.5d7d2f00.js to remote file: /webroot/hade_doc/dist/assets/js/20.5d7d2f00.js finish"	map[]
[Info]	2022-12-14T20:37:48+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/21.c5cbe54a.js to remote file: /webroot/hade_doc/dist/assets/js/21.c5cbe54a.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/22.0150d521.js to remote file: /webroot/hade_doc/dist/assets/js/22.0150d521.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/23.58cff40f.js to remote file: /webroot/hade_doc/dist/assets/js/23.58cff40f.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/24.562746b5.js to remote file: /webroot/hade_doc/dist/assets/js/24.562746b5.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/25.7d55bc4b.js to remote file: /webroot/hade_doc/dist/assets/js/25.7d55bc4b.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/26.fab7722a.js to remote file: /webroot/hade_doc/dist/assets/js/26.fab7722a.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/27.ae482208.js to remote file: /webroot/hade_doc/dist/assets/js/27.ae482208.js finish"	map[]
[Info]	2022-12-14T20:37:49+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/28.5f461182.js to remote file: /webroot/hade_doc/dist/assets/js/28.5f461182.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/29.035f9ced.js to remote file: /webroot/hade_doc/dist/assets/js/29.035f9ced.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/3.928ce6a6.js to remote file: /webroot/hade_doc/dist/assets/js/3.928ce6a6.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/4.f177e320.js to remote file: /webroot/hade_doc/dist/assets/js/4.f177e320.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/5.529ffd1a.js to remote file: /webroot/hade_doc/dist/assets/js/5.529ffd1a.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/6.a92ad380.js to remote file: /webroot/hade_doc/dist/assets/js/6.a92ad380.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/7.47c47502.js to remote file: /webroot/hade_doc/dist/assets/js/7.47c47502.js finish"	map[]
[Info]	2022-12-14T20:37:50+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/8.8ae34656.js to remote file: /webroot/hade_doc/dist/assets/js/8.8ae34656.js finish"	map[]
[Info]	2022-12-14T20:37:51+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/9.cc18f0e9.js to remote file: /webroot/hade_doc/dist/assets/js/9.cc18f0e9.js finish"	map[]
[Info]	2022-12-14T20:37:51+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/assets/js/app.5ce4c1ae.js to remote file: /webroot/hade_doc/dist/assets/js/app.5ce4c1ae.js finish"	map[]
[Info]	2022-12-14T20:37:51+08:00	"mkdir: /webroot/hade_doc/dist/guide"	map[]
[Info]	2022-12-14T20:37:51+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/app.html to remote file: /webroot/hade_doc/dist/guide/app.html finish"	map[]
[Info]	2022-12-14T20:37:51+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/build.html to remote file: /webroot/hade_doc/dist/guide/build.html finish"	map[]
[Info]	2022-12-14T20:37:51+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/command.html to remote file: /webroot/hade_doc/dist/guide/command.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/cron.html to remote file: /webroot/hade_doc/dist/guide/cron.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/dev.html to remote file: /webroot/hade_doc/dist/guide/dev.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/env.html to remote file: /webroot/hade_doc/dist/guide/env.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/index.html to remote file: /webroot/hade_doc/dist/guide/index.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/install.html to remote file: /webroot/hade_doc/dist/guide/install.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/introduce.html to remote file: /webroot/hade_doc/dist/guide/introduce.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/middleware.html to remote file: /webroot/hade_doc/dist/guide/middleware.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/model.html to remote file: /webroot/hade_doc/dist/guide/model.html finish"	map[]
[Info]	2022-12-14T20:37:52+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/provider.html to remote file: /webroot/hade_doc/dist/guide/provider.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/structure.html to remote file: /webroot/hade_doc/dist/guide/structure.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/swagger.html to remote file: /webroot/hade_doc/dist/guide/swagger.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/todo.html to remote file: /webroot/hade_doc/dist/guide/todo.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/util.html to remote file: /webroot/hade_doc/dist/guide/util.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/guide/version.html to remote file: /webroot/hade_doc/dist/guide/version.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/index.html to remote file: /webroot/hade_doc/dist/index.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"mkdir: /webroot/hade_doc/dist/provider"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/provider/app.html to remote file: /webroot/hade_doc/dist/provider/app.html finish"	map[]
[Info]	2022-12-14T20:37:53+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/provider/config.html to remote file: /webroot/hade_doc/dist/provider/config.html finish"	map[]
[Info]	2022-12-14T20:37:54+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/provider/env.html to remote file: /webroot/hade_doc/dist/provider/env.html finish"	map[]
[Info]	2022-12-14T20:37:54+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/provider/index.html to remote file: /webroot/hade_doc/dist/provider/index.html finish"	map[]
[Info]	2022-12-14T20:37:54+08:00	"upload local file: /Users/jianfengye/Documents/workspace/gohade/hade/deploy/20221214203745/dist/provider/log.html to remote file: /webroot/hade_doc/dist/provider/log.html finish"	map[]
[Info]	2022-12-14T20:37:54+08:00	"upload folder success"	map[]
[Info]	2022-12-14T20:37:54+08:00	"execute post action start"	map[cmd:pwd connection:ssh.web-01]
[Info]	2022-12-14T20:37:54+08:00	"execute post action finish"	map[cmd:pwd connection:ssh.web-01 out:/home/yejianfeng]
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h3 id="部署后端" tabindex="-1"><a class="header-anchor" href="#部署后端" aria-hidden="true">#</a> 部署后端</h3>
<p>命令 <code v-pre>./hade deploy backend</code>
会自动编译hade二进制文件，然后上传到服务器上。
如果你的post_action 设置的是重启远端服务器进程，那么实际上就是一个完整的cd行为了。</p>
<h3 id="前后端一起部署" tabindex="-1"><a class="header-anchor" href="#前后端一起部署" aria-hidden="true">#</a> 前后端一起部署</h3>
<p>命令 <code v-pre>./hade deploy all</code></p>
<h3 id="部署回滚" tabindex="-1"><a class="header-anchor" href="#部署回滚" aria-hidden="true">#</a> 部署回滚</h3>
<p>每次部署执行，都会在本地的deploy目录下创建一个目录，目录名为当前时间戳，比如<code v-pre>20221214203745</code>。</p>
<p>如果你想回滚到上一次部署的版本，可以执行命令 <code v-pre>./hade deploy rollback 20221214203745 backend</code>。</p>
<p>实际上做的事情就是将deploy目录下的时间戳对应的文件再进行一次发布。</p>
</div></template>


