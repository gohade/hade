<template><div><h1 id="服务提供者" tabindex="-1"><a class="header-anchor" href="#服务提供者" aria-hidden="true">#</a> 服务提供者</h1>
<h2 id="指南" tabindex="-1"><a class="header-anchor" href="#指南" aria-hidden="true">#</a> 指南</h2>
<p>hade框架使用ServiceProvider机制来满足协议，通过service Provder提供某个协议服务的具体实现。这样如果开发者对具体的实现协议的服务类的具体实现不满意，则可以很方便的通过切换具体协议的Service Provider来进行具体服务的切换。</p>
<p>一个ServiceProvider是一个单独的文件夹，它包含服务提供和服务实现。具体可以参考framework/provider/demo</p>
<p>一个SerivceProvider就是一个独立的包，这个包可以作为插件独立地发布和分享。</p>
<p>你也可以定义一个无contract的ServiceProvider，其中的Name()需要保证唯一。</p>
<h2 id="创建" tabindex="-1"><a class="header-anchor" href="#创建" aria-hidden="true">#</a> 创建</h2>
<p>我们可以使用命令 <code v-pre>./hade provider new</code> 来创建一个新的service provider</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade provider new
create a provider
? please input provider name test
? please input provider folder(default: provider name):
create provider success, folder path: /Users/Documents/workspace/hade_workspace/demo5/app/provider/test
please remember add provider to kernel
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>该命令会在<code v-pre>app/provider/</code> 目录下创建一个对应的服务提供者。并且初始化好三个文件： <code v-pre>contract.go</code>, <code v-pre>provider.go</code>, <code v-pre>service.go</code></p>
<h2 id="自定义" tabindex="-1"><a class="header-anchor" href="#自定义" aria-hidden="true">#</a> 自定义</h2>
<p>我们需要编写这三个文件：</p>
<h3 id="contract-go" tabindex="-1"><a class="header-anchor" href="#contract-go" aria-hidden="true">#</a> contract.go</h3>
<p>contract.go 定义了这个服务提供方提供的协议接口。hade 框架任务，作为一个业务的服务提供者，定义一个好的协议是最重要的事情。</p>
<p>所以 contract.go 中定义了一个 Service 接口，在其中定义各种方法，包含输入参数和返回参数。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>package demo

const DemoKey = "demo"

type IService interface {
	GetAllStudent() []Student
}

type Student struct {
	ID   int
	Name string
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>其中还定义了一个Key， 这个 Key 是全应用唯一的，服务提供者将服务以 Key 关键字注入到容器中。服务使用者使用 Key 关键字获取服务。</p>
<h3 id="provider" tabindex="-1"><a class="header-anchor" href="#provider" aria-hidden="true">#</a> provider</h3>
<p>provider.go 提供服务适配的实现，实现一个Provider必须实现对应的五个方法</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>package demo

import (
	"github.com/gohade/hade/framework"
)

type DemoProvider struct {
	framework.ServiceProvider

	c framework.Container
}

func (sp *DemoProvider) Name() string {
	return DemoKey
}

func (sp *DemoProvider) Register(c framework.Container) framework.NewInstance {
	return NewService
}

func (sp *DemoProvider) IsDefer() bool {
	return false
}

func (sp *DemoProvider) Params() []interface{} {
	return []interface{}{sp.c}
}

func (sp *DemoProvider) Boot(c framework.Container) error {
	sp.c = c
	return nil
}
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><ul>
<li>Name() // 指定这个服务提供者提供的服务对应的接口的关键字</li>
<li>Register() // 这个服务提供者注册的时候调用的方法，一般是指定初始化服务的函数名</li>
<li>IsDefer() // 这个服务是否是使用时候再进行初始化，false为注册的时候直接进行初始化服务</li>
<li>Params() // 初始化服务的时候对服务注入什么参数，一般把 container 注入到服务中</li>
<li>Boot() // 初始化之前调用的函数，一般设置一些全局的Provider</li>
</ul>
<h3 id="service-go" tabindex="-1"><a class="header-anchor" href="#service-go" aria-hidden="true">#</a> service.go</h3>
<p>service.go提供具体的实现，它至少需要提供一个实例化的方法 <code v-pre>NewService(params ...interface{}) (interface{}, error)</code>。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>package demo

import "github.com/gohade/hade/framework"

type Service struct {
	container framework.Container
}

func NewService(params ...interface{}) (interface{}, error) {
	container := params[0].(framework.Container)
	return &amp;Service{container: container}, nil
}

func (s *Service) GetAllStudent() []Student {
	return []Student{
		{
			ID:   1,
			Name: "foo",
		},
		{
			ID:   2,
			Name: "bar",
		},
	}
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="注入" tabindex="-1"><a class="header-anchor" href="#注入" aria-hidden="true">#</a> 注入</h2>
<p>hade的路由，controller的定义是选择基于gin框架进行扩展的。所有的gin框架的路由，参数获取，验证，context都和gin框架是相同的。唯一不同的是gin的全局路由gin.Engine实现了hade的容器结构，可以对gin.Engine进行服务提供的注入，且可以从context中获取具体的服务。</p>
<p>hade提供两种服务注入的方法：</p>
<ul>
<li>Bind: 将一个ServiceProvider绑定到容器中，可以控制其是否是单例</li>
<li>Singleton: 将一个单例ServiceProvider绑定到容器中</li>
</ul>
<p>建议在文件夹 <code v-pre>app/provider/kernel.go</code> 中进行服务注入</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>func RegisterCustomProvider(c framework.Container) {
	c.Bind(&amp;demo.DemoProvider{}, true)
}
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>当然你也可以在某个业务模块路由注册的时候进行服务注入</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>func Register(r *gin.Engine) error {
	api := NewDemoApi()
	r.Container().Singleton(&amp;demoService.DemoProvider{})

	r.GET(&quot;/demo/demo&quot;, api.Demo)
	r.GET(&quot;/demo/demo2&quot;, api.Demo2)
	return nil
}
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="获取" tabindex="-1"><a class="header-anchor" href="#获取" aria-hidden="true">#</a> 获取</h2>
<p>hade提供了三种服务获取的方法：</p>
<ul>
<li>Make: 根据一个Key获取服务，获取不到获取报错</li>
<li>MustMake: 根据一个Key获取服务，获取不到返回空</li>
<li>MakeNew: 根据一个Key获取服务，每次获取都实例化，对应的ServiceProvider必须是以非单例形式注入</li>
</ul>
<p>你可以在任意一个可以获取到 container 的地方进行服务的获取。</p>
<p>业务模块中:</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>func (api *DemoApi) Demo2(c *gin.Context) {
	demoProvider := c.MustMake(demoService.DemoKey).(demoService.IService)
	students := demoProvider.GetAllStudent()
	usersDTO := StudentsToUserDTOs(students)
	c.JSON(200, usersDTO)
}
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>命令行中：</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>var CenterCommand = &amp;cobra.Command{
	Use:   &quot;direct_center&quot;,
	Short: &quot;计算区域中心点&quot;,
	RunE: func(c *cobra.Command, args []string) error {
		container := util.GetContainer(c.Root())
		app := container.MustMake(contract.AppKey).(contract.App)
        return nil
    }
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>甚至于另外一个服务提供者中：</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>type Service struct {
	c framework.Container

	baseURL string
	userID  string
	token   string
	logger  contract.Log
}

func NewService(params ...interface{}) (interface{}, error) {
	c := params[0].(framework.Container)
	config := c.MustMake(contract.ConfigKey).(contract.Config)
	baseURL := config.GetString(&quot;app.stsmap.url&quot;)
	userID := config.GetString(&quot;app.stsmap.user_id&quot;)
	token := config.GetString(&quot;app.stsmap.token&quot;)

	logger := c.MustMake(contract.LogKey).(contract.Log)
	return &amp;Service{baseURL: baseURL, logger: logger, userID: userID, token: token}, nil
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><h2 id="hade-provider" tabindex="-1"><a class="header-anchor" href="#hade-provider" aria-hidden="true">#</a> hade provider</h2>
<p>hade 框架默认自带了一些服务提供者，提供基础的服务接口协议，可以通过 <code v-pre>./hade provider list</code> 来获取已经安装的服务提供者。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade provider list
hade:app
hade:env
hade:config
hade:log
hade:ssh
hade:kernel
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>hade 框架自带的服务提供者的 key 是以 <code v-pre>hade:</code> 开头。目的为的是与自定义服务提供者的 key 区别开。</p>
<p>hade 框架自带的服务提供者具体定义的协议可以参考：<RouterLink to="/provider/">provider</RouterLink></p>
</div></template>


