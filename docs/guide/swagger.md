# swagger

## 命令

hade 使用 [swaggo](https://github.com/swaggo/swag) 集成了 swagger 生成和服务项目。并且封装了 `./hade swagger` 命令。

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade swagger
swagger operator

Usage:
  hade swagger [flags]
  hade swagger [command]

Available Commands:
  gen         generate swagger file, contain swagger.yaml, doc.go
  serve       use gen serve

Flags:
  -h, --help   help for swagger

Use "hade swagger [command] --help" for more information about a command.
```

- gen  // 生成swagger文件
- serve // 提供swagger服务

## 注释

hade 使用 [swaggo](https://github.com/swaggo/swag) 来实现注释生成 swagger 功能。

全局注释在文件  `app/http/swagger.go` 中:
```
// Package http API.
// @title hade
// @version 1.0
// @description hade测试
// @termsOfService https://github.com/swaggo/swag

// @contact.name yejianfeng
// @contact.email yejianfeng

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8989
// @BasePath /api
// @query.collection.format multi

// @securityDefinitions.basic BasicAuth

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @x-extension-openapi {"example": "value on a json format"}

package http

```

接口注释请写在各自模块的 api.go 中

```
// Demo godoc
// @Summary 获取所有用户
// @Description 获取所有用户
// @Produce  json
// @Tags demo
// @Success 200 array []UserDTO
// @Router /demo/demo [get]
func (api *DemoApi) Demo(c *gin.Context) {
	users := api.service.GetUsers()
	usersDTO := UserModelsToUserDTOs(users)
	c.JSON(200, usersDTO)
}
```

swagger 注释的格式和关键词可以参考：[swaggo](https://github.com/swaggo/swag)

## 生成

使用命令 `./hade swagger gen`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade swagger gen
2020/09/16 19:57:33 Generate swagger docs....
2020/09/16 19:57:33 Generate general API Info, search dir:./app/http/
2020/09/16 19:57:33 Generating demo.UserDTO
2020/09/16 19:57:33 create docs.go at /Users/Documents/workspace/hade_workspace/demo5/app/http/swagger/docs.go
2020/09/16 19:57:33 create swagger.json at /Users/Documents/workspace/hade_workspace/demo5/app/http/swagger/swagger.json
2020/09/16 19:57:33 create swagger.yaml at /Users/Documents/workspace/hade_workspace/demo5/app/http/swagger/swagger.yaml
```

在目录 `app/http/swagger/` 下自动生成swagger相关文件。

## 服务

可以使用命令 `./hade swagger serve` 启动当前应用的 swagger ui 服务。

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade swagger serve
swagger success: http://0.0.0.0:8069/swagger/index.html
if you want to replace, remember use command: swagger gen
```

::: tip
如果你的 swagger 服务已经启动，更新 swagger 只需要重新运行 `./hade swagger gen` 就能更新。因为 swagger 服务读取的是生成的 swagger.json 这个文件。
:::

服务端口，我们也可以通过配置文件 `config/[env]/swagger.yaml` 中的配置 
```
url: http://127.0.0.1:8069
```

来配置swagger serve 启动的服务。