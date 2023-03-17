<template><div><h1 id="目录结构" tabindex="-1"><a class="header-anchor" href="#目录结构" aria-hidden="true">#</a> 目录结构</h1>
<p>hade 框架不仅仅是一个类库，也是一个定义了开发模式和目录结构的框架。 hade 希望所有使用这个框架的开发人员遵照统一的项目结构进行开发。</p>
<h2 id="默认目录结构" tabindex="-1"><a class="header-anchor" href="#默认目录结构" aria-hidden="true">#</a> 默认目录结构</h2>
<p>默认创建的项目结构为：</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ tree
.
├── README.md
├── app // 服务端应用地址
│   ├── console  // 存放自定义命令
│   │   ├── command
│   │   │   └── demo.go
│   │   └── kernel.go
│   ├── http  // 存放http服务
│   │   ├── helper // 帮助类库
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   ├── kernel.go
│   │   ├── middleware // 中间件
│   │   ├── module // 业务模块
│   │   │   └── demo 
│   │   │       ├── api.go // 业务模块接口
│   │   │       ├── dto.go // 业务模块输出结构
│   │   │       ├── mapper.go // 将服务结构转换为业务模块输出结构
│   │   │       ├── model.go // 数据库结构定义
│   │   │       ├── repository.go // 数据库逻辑封装层
│   │   │       └── service.go // 服务层
│   │   ├── route.go // 路由配置
│   │   ├── swagger // swagger文件自动生成 
│   │   └── swagger.go
│   └── provider // 服务提供方
│       ├── demo
│       │   ├── contract.go // 服务接口层
│       │   ├── provider.go // 服务提供方
│       │   └── service.go // 服务实现层
│       └── kernel.go // 服务提供方注入
├── babel.config.js
├── config // 配置文件
│   ├── development // development环境
│   │   ├── app.yaml // app主应用的配置
│   │   ├── database.yaml // 数据库相关配置
│   │   ├── deploy.yaml // 部署相关配置
│   │   ├── log.yaml // 日志相关配置
│   │   └── swagger.yaml // swagger相关配置
│   ├── production
│   └── testing
├── dist // 编译生成地址
├── frontend // 前端应用地址
│   ├── App.vue // vue入口文件
│   ├── assets
│   │   └── logo.png
│   ├── components // vue组件
│   │   └── HelloWorld.vue
│   └── main.js // 前端入口文件
├── go.mod
├── go.sum
├── main.go // app入口
├── package.json // 前端package
├── storage // 存储目录
│   ├── cache // 存放本地缓存
│   ├── coverage // 存放覆盖率报告
│   ├── log // 存放业务日志
│   └── pid // 存放pid
│       └── app.pid
├── tests // 测试相关数据
│   └── env.go // 设置测试环境相关参数
└── vue.config.js // vue配置

25 directories, 37 files
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>这里主要介绍下业务模块的分层结构</p>
<h1 id="业务模块分层" tabindex="-1"><a class="header-anchor" href="#业务模块分层" aria-hidden="true">#</a> 业务模块分层</h1>
<p>业务模块的分层设计两种分层模型：简化模型和标准模型。基本稍微复杂一些的业务，都需要使用标准模型开发。</p>
<h2 id="简化模型" tabindex="-1"><a class="header-anchor" href="#简化模型" aria-hidden="true">#</a> 简化模型</h2>
<p>对于比较简单的业务，每个模块各自定义自己的 model 和 service，在一个 module 文件的文件夹中进行各自模块的业务开发</p>
<p><img src="http://tuchuang.funaio.cn/md/20200916111330.png" alt="20200916111330"></p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>├── api.go // 业务模块接口
├── dto.go // 业务模块输出结构
├── mapper.go // 将服务结构转换为业务模块输出结构
├── model.go // 数据库结构定义
├── repository.go // 数据库逻辑封装层
└── service.go // 服务
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>具体实现可以参考初始化代码的 Demo 接口实现</p>
<h2 id="标准模型" tabindex="-1"><a class="header-anchor" href="#标准模型" aria-hidden="true">#</a> 标准模型</h2>
<p>对于比较复杂的业务，模块与模块间的交互比较复杂，有很多公用性，所以提取 service provider 服务作为服务间的相互调用。</p>
<div class="custom-container tip"><p class="custom-container-title">TIP</p>
<p>强烈建议使用这种开发模型</p>
</div>
<p><img src="http://tuchuang.funaio.cn/md/20200916111454.png" alt="20200916111454"></p>
<p>第一步：创建当前业务的 provider。可以使用命令行 <code v-pre>./hade provider new</code> 来创建。</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>[~/Documents/workspace/hade_workspace/demo5]$ ./hade provider new
create a provider
? please input provider name car
? please input provider folder(default: provider name):
create provider success, folder path: /Users/xxx/Documents/workspace/hade_workspace/demo5/app/provider/car
please remember add provider to kernel
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>定义好 provider 的协议</p>
<div class="language-text line-numbers-mode" data-ext="text"><pre v-pre class="language-text"><code>package demo

const DemoKey = "demo"

type IService interface {
	GetAllStudent() []Student
}

type Student struct {
	ID   int
	Name string
}

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>实现对应协议：</p>
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

</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>第二步：创建当前业务的模块。</p>
<p>可以按照demo文件夹中文件编写。</p>
<p><img src="http://tuchuang.funaio.cn/md/20200916172527.png" alt="20200916172527"></p>
<p>第三步：在当前业务中挂载业务模块。</p>
<p><img src="http://tuchuang.funaio.cn/md/20200916172752.png" alt="20200916172752"></p>
<p>第四步：使用 provider 来开发当前业务。</p>
<div class="language-golang line-numbers-mode" data-ext="golang"><pre v-pre class="language-golang"><code>// Demo godoc
// @Summary 获取所有学生
// @Description 获取所有学生
// @Produce  json
// @Tags demo
// @Success 200 array []UserDTO
// @Router /demo/demo2 [get]
func (api *DemoApi) Demo2(c *gin.Context) {
	demoProvider := c.MustMake(demoService.DemoKey).(demoService.IService)
	students := demoProvider.GetAllStudent()
	usersDTO := StudentsToUserDTOs(students)
	c.JSON(200, usersDTO)
}
</code></pre><div class="line-numbers" aria-hidden="true"><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div><div class="line-number"></div></div></div><p>具体实现可以参考初始化代码的 Demo2 接口实现</p>
</div></template>


