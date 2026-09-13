# 运行

## 命令

这里的运行是运行整个 app，这个 app 可以只包含后端，也可以只包含前端，但是后端也是隐藏在前端后面运行。具体可以参考
app/http/route.go

```
package http

import (
	"github.com/gohade/hade/app/http/controller/demo"
	"github.com/gohade/hade/framework/gin"
)

func Routes(r *gin.Engine) {
	r.Static("/dist/", "./dist/")
	r.GET("/demo/demo", demo.Demo)
}

```

运行相关的命令为 app。

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade app
start app serve

Usage:
  hade app [flags]
  hade app [command]

Available Commands:
  restart     restart app server
  start       start app server
  state       get app pid
  stop        stop app server

Flags:
  -h, --help   help for app

Use "hade app [command] --help" for more information about a command.
```

## 启动

可以使用 `./hade app start` 启动一个应用。

```markdown
成功启动进程: hade app
进程pid: 39327
监听地址: http://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
```

也可以使用 `./hade app start -d` 使用 deamon 模式启动一个应用。应用名称为 `hade app`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade app start -d
成功启动进程: hade app
进程pid: 41021
监听地址: http://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/teststorage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
```

app 应用的输出记录在 `/storage/log/app.log`

进程 id 记录在 `/storage/pid/app.pid`

## 状态

当使用 deamon 模式启动的时候，需要查看当前应用是否有启动，如果启动了，进程号是多少，可以使用命令 `./hade app state`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade app state
app服务已经启动, pid: 41021
```

## 重启

当使用 deamon 模式启动的时候，需要重启应用，可以使用命令 `./hade app restart`

::: tip
如果程序还未启动，调用 restart 命令，效果和 start 命令一样，deamon 模式启动应用
:::

## 停止

当使用 deamon 模式启动的时候，需要关闭应用，可以使用命令 `./hade app stop`

## 进程运行基础配置

在启动进程的时候，我们会需要定义一些配置项，这些配置项决定进程的运行环境（比如日志存放位置，运行态信息存放位置，配置文件存放位置等）。

这里我们提供了3种配置方式来设置这些基础配置，包括环境变量设置，命令行参数设置，配置文件设置。

这三种配置方式的优先级为：命令行参数 > 环境变量 > 配置文件

具体的配置项常用的如下，具体更多可以参考 framework/provider/app/service.go ：

* 运行中间信息存放目录
    * 命令行参数：--runtime_folder
    * 环境变量：RUNTIME_FOLDER
    * 配置文件：app.path.runtime_folder
    * 不设置默认为：运行信息基础目录 + runtime
* 日志存放目录
    * 命令行参数：--log_folder
    * 环境变量：LOG_FOLDER
    * 配置文件：app.path.log_folder
    * 不设置默认为：运行信息基础目录 + log
* 运行信息基础目录
    * 命令行参数：--storage_folder
    * 环境变量：STORAGE_FOLDER
    * 配置文件：app.path.storage_folder
    * 不设置默认为：基础目录 + storage
* 配置文件地址
    * 命令行参数：--config_folder
    * 环境变量：CONFIG_FOLDER
    * 配置文件：app.path.config_folder
    * 不设置默认为：基础目录 + config
* 基础目录
    * 命令行参数：--base_folder
    * 环境变量：BASE_FOLDER
    * 配置文件：app.path.base_folder
    * 不设置默认为：当前执行目录

### 环境变量设置

在启动进程的时候进行环境变量的设置。比如

```markdown
STORAGE_FOLDER=/Users/jianfengye/Documents/workspace/gohade/hade/teststorage ./hade app start
```

### 命令行参数设置

在命令行参数中设置。比如

```markdown
./hade app start --storage_folder=/Users/jianfengye/Documents/workspace/gohade/hade/teststorage/ -d
```

### 配置文件设置

在配置文件config/${env}/app.yaml中配置：

```markdown
path:
storage_folder: "/Users/jianfengye/Documents/workspace/gohade/hade/teststorage/"
```

