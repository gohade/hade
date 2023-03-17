<template><div><h1 id="模型" tabindex="-1"><a class="header-anchor" href="#模型" aria-hidden="true">#</a> 模型</h1>
<h2 id="指南" tabindex="-1"><a class="header-anchor" href="#指南" aria-hidden="true">#</a> 指南</h2>
<p>hade model 提供了自动生成数据库模型的命令行工具，如果你已经定义好你的model，这里的系列工具能帮助你节省不少时间。</p>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code><span class="token operator">></span> ./hade model
数据库模型相关的命令

Usage:
  hade model <span class="token punctuation">[</span>flags<span class="token punctuation">]</span>
  hade model <span class="token punctuation">[</span>command<span class="token punctuation">]</span>

Available Commands:
  api         通过数据库生成api
  gen         生成模型
  <span class="token builtin class-name">test</span>        测试数据库

Flags:
  -h, <span class="token parameter variable">--help</span>   <span class="token builtin class-name">help</span> <span class="token keyword">for</span> model
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>包含三个命令：</p>
<ul>
<li>./hade model api 通过数据表自动生成api代码</li>
<li>./hade model gen 通过数据表自动生成model代码</li>
<li>./hade model test 测试某个数据库是否可以连接</li>
</ul>
<h2 id="hade-model-test" tabindex="-1"><a class="header-anchor" href="#hade-model-test" aria-hidden="true">#</a> ./hade model test</h2>
<p>当你想测试下你的某个配置好的数据库是否能连接上，都有哪些表的时候，这个命令能帮助你。</p>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code><span class="token operator">></span> ./hade model <span class="token builtin class-name">test</span> <span class="token parameter variable">--database</span><span class="token operator">=</span><span class="token string">"database.default"</span>
数据库连接：database.default成功
一共存在1张表
student
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="hade-model-gen" tabindex="-1"><a class="header-anchor" href="#hade-model-gen" aria-hidden="true">#</a> ./hade model gen</h2>
<p>这个命令帮助你生成数据表对应的gorm的model</p>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code> ✘  ~/Documents/workspace/gohade/hade   feature/model-gen ●  ./hade model gen
Error: required flag<span class="token punctuation">(</span>s<span class="token punctuation">)</span> <span class="token string">"output"</span> not <span class="token builtin class-name">set</span>
Usage:
  hade model gen <span class="token punctuation">[</span>flags<span class="token punctuation">]</span>

Flags:
  -d, <span class="token parameter variable">--database</span> string   模型连接的数据库 <span class="token punctuation">(</span>default <span class="token string">"database.default"</span><span class="token punctuation">)</span>
  -h, <span class="token parameter variable">--help</span>              <span class="token builtin class-name">help</span> <span class="token keyword">for</span> gen
  -o, <span class="token parameter variable">--output</span> string     模型输出地址

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>其中接受两个参数：</p>
<p>database: 这个参数可选，用来表示模型连接的数据库配置地址，默认是database.default，表示config目录下的{env}目录下的database.yaml中的default配置。</p>
<p>output：这个参数必填，用来表示模型文件的输出地址，如果填写相对路径，会在前面填充当前执行路径来补充为绝对路径。</p>
<h3 id="使用方式" tabindex="-1"><a class="header-anchor" href="#使用方式" aria-hidden="true">#</a> 使用方式</h3>
<p>第一步，使用 <code v-pre>./hade model gen --output=app/model</code></p>
<p><img src="http://tuchuang.funaio.cn/img/image-20220215091522181.png" alt="image-20220215091522181"></p>
<p>选择其中的两个表，answers和questions，提示目录文件</p>
<p><img src="http://tuchuang.funaio.cn/img/image-20220215091541908.png" alt="image-20220215091541908"></p>
<p>下一步确认y继续</p>
<p><img src="http://tuchuang.funaio.cn/img/image-20220215091643242.png" alt="image-20220215091643242"></p>
<p>最后生成模型成功</p>
<p><img src="http://tuchuang.funaio.cn/img/image-20220215091659104.png" alt="image-20220215091659104"></p>
<p>查看文件，确实生成了model</p>
<p><img src="http://tuchuang.funaio.cn/img/image-20220215091735434.png" alt="image-20220215091735434"></p>
<h2 id="hade-model-api" tabindex="-1"><a class="header-anchor" href="#hade-model-api" aria-hidden="true">#</a> ./hade model api</h2>
<div class="language-bash line-numbers-mode" data-ext="sh"><pre v-pre class="language-bash"><code><span class="token operator">></span> ./hade model api <span class="token parameter variable">--database</span><span class="token operator">=</span>database.default <span class="token parameter variable">--output</span><span class="token operator">=</span>/Users/jianfengye/Documents/workspace/gohade/hade/app/http/module/student/
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div></div></div><p><img src="http://tuchuang.funaio.cn/markdown/202303140005210.png" alt=""></p>
<p>它会在目标文件夹中生成对这个数据表的5个接口文件</p>
<ul>
<li>gen_[table]_api_create.go</li>
<li>gen_[table]_api_delete.go</li>
<li>gen_[table]_api_list.go</li>
<li>gen_[table]_api_show.go</li>
<li>gen_[table]_api_update.go</li>
</ul>
<p>还有另外两个文件</p>
<ul>
<li>gen_[table]_model.go // 数据表对应的模型结构</li>
<li>gen_[table]_router.go // 5个接口对应的默认路由</li>
</ul>
<blockquote>
<p>这里的每个文件都是可以修改的。</p>
<p>但是注意，如果重新生成，会覆盖原先的文件。</p>
<p>执行命令的时候切记保存已经改动的代码文件。</p>
</blockquote>
</div></template>


