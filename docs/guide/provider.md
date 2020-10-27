# 服务提供者

## 指南

hade框架使用ServiceProvider机制来满足协议，通过service Provder提供某个协议服务的具体实现。这样如果开发者对具体的实现协议的服务类的具体实现不满意，则可以很方便的通过切换具体协议的Service Provider来进行具体服务的切换。

一个ServiceProvider是一个单独的文件夹，它包含服务提供和服务实现。具体可以参考framework/provider/demo

一个SerivceProvider就是一个独立的包，这个包可以作为插件独立地发布和分享。

你也可以定义一个无contract的ServiceProvider，其中的Name()需要保证唯一。

## 创建

我们可以使用命令 `./hade provider new` 来创建一个新的service provider

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade provider new
create a provider
? please input provider name test
? please input provider folder(default: provider name):
create provider success, folder path: /Users/Documents/workspace/hade_workspace/demo5/app/provider/test
please remember add provider to kernel
```

该命令会在`app/provider/` 目录下创建一个对应的服务提供者。并且初始化好三个文件： `contract.go`, `provider.go`, `service.go`

## 自定义

我们需要编写这三个文件：

### contract.go

contract.go 定义了这个服务提供方提供的协议接口。hade 框架任务，作为一个业务的服务提供者，定义一个好的协议是最重要的事情。

所以 contract.go 中定义了一个 Service 接口，在其中定义各种方法，包含输入参数和返回参数。

```
package demo

const DemoKey = "demo"

type IService interface {
	GetAllStudent() []Student
}

type Student struct {
	ID   int
	Name string
}

```

其中还定义了一个Key， 这个 Key 是全应用唯一的，服务提供者将服务以 Key 关键字注入到容器中。服务使用者使用 Key 关键字获取服务。

### provider

provider.go 提供服务适配的实现，实现一个Provider必须实现对应的五个方法
```
package demo

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
```

- Name() // 指定这个服务提供者提供的服务对应的接口的关键字
- Register() // 这个服务提供者注册的时候调用的方法，一般是指定初始化服务的函数名
- IsDefer() // 这个服务是否是使用时候再进行初始化，false为注册的时候直接进行初始化服务
- Params() // 初始化服务的时候对服务注入什么参数，一般把 container 注入到服务中
- Boot() // 初始化之前调用的函数，一般设置一些全局的Provider


### service.go

service.go提供具体的实现，它至少需要提供一个实例化的方法 `NewService(params ...interface{}) (interface{}, error)`。

```
package demo

import "github.com/gohade/hade/framework"

type Service struct {
	container framework.Container
}

func NewService(params ...interface{}) (interface{}, error) {
	container := params[0].(framework.Container)
	return &Service{container: container}, nil
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

```

## 注入

hade的路由，controller的定义是选择基于gin框架进行扩展的。所有的gin框架的路由，参数获取，验证，context都和gin框架是相同的。唯一不同的是gin的全局路由gin.Engine实现了hade的容器结构，可以对gin.Engine进行服务提供的注入，且可以从context中获取具体的服务。

hade提供两种服务注入的方法：
* Bind: 将一个ServiceProvider绑定到容器中，可以控制其是否是单例
* Singleton: 将一个单例ServiceProvider绑定到容器中

建议在文件夹 `app/provider/kernel.go` 中进行服务注入

``` golang 
func RegisterCustomProvider(c framework.Container) {
	c.Bind(&demo.DemoProvider{}, true)
}
```

当然你也可以在某个业务模块路由注册的时候进行服务注入

``` golang
func Register(r *gin.Engine) error {
	api := NewDemoApi()
	r.Container().Singleton(&demoService.DemoProvider{})

	r.GET("/demo/demo", api.Demo)
	r.GET("/demo/demo2", api.Demo2)
	return nil
}
```

## 获取

hade提供了三种服务获取的方法：
* Make: 根据一个Key获取服务，获取不到获取报错
* MustMake: 根据一个Key获取服务，获取不到返回空
* MakeNew: 根据一个Key获取服务，每次获取都实例化，对应的ServiceProvider必须是以非单例形式注入

你可以在任意一个可以获取到 container 的地方进行服务的获取。

业务模块中:

```
func (api *DemoApi) Demo2(c *gin.Context) {
	demoProvider := c.MustMake(demoService.DemoKey).(demoService.IService)
	students := demoProvider.GetAllStudent()
	usersDTO := StudentsToUserDTOs(students)
	c.JSON(200, usersDTO)
}
```

命令行中：

``` golang
var CenterCommand = &cobra.Command{
	Use:   "direct_center",
	Short: "计算区域中心点",
	RunE: func(c *cobra.Command, args []string) error {
		container := util.GetContainer(c.Root())
		app := container.MustMake(contract.AppKey).(contract.App)
        return nil
    }
```

甚至于另外一个服务提供者中：

``` golang 
type Service struct {
	c framework.Container

	baseURL string
	userID  string
	token   string
	logger  contract.Log
}

func NewService(params ...interface{}) (interface{}, error) {
	c := params[0].(framework.Container)
	config := c.MustMake(contract.ConfigKey).(contract.Config)
	baseURL := config.GetString("app.stsmap.url")
	userID := config.GetString("app.stsmap.user_id")
	token := config.GetString("app.stsmap.token")

	logger := c.MustMake(contract.LogKey).(contract.Log)
	return &Service{baseURL: baseURL, logger: logger, userID: userID, token: token}, nil
}

```

## hade provider

hade 框架默认自带了一些服务提供者，提供基础的服务接口协议，可以通过 `./hade provider list` 来获取已经安装的服务提供者。

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade provider list
hade:app
hade:env
hade:config
hade:log
hade:ssh
hade:kernel
```

hade 框架自带的服务提供者的 key 是以 `hade:` 开头。目的为的是与自定义服务提供者的 key 区别开。

hade 框架自带的服务提供者具体定义的协议可以参考：[provider](/provider/)