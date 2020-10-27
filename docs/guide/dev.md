# 调试模式

## 命令

hade 框架自带调试模式，不管是前端还是后端，都可以启动调试模式，边修改代码，边编译运行服务。

对应的命令为 `./hade dev`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev
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
```

- 调试前端
- 调试后端
- 同时调试

## 调试前端

使用命令 `./hade dev frontend`

要求当前编译机器安装 npm 软件，并且当前项目已经运行了 npm install，安装完成前端依赖。

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev frontend

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
```

实际上是调用 `npm run dev` 来调试前端

## 调试后端

使用命令 `./hade dev backend`

要求当前编译机器安装 go 软件，版本 > 1.3。

```
[~/Documents/workspace/hade_workspace/demo5]$  ./hade dev backend
./hade dev backend
build success please run ./hade direct
backend server: http://127.0.0.1:15060
proxy backend server: http://0.0.0.0:8073
[PID] 29034
app serve url: http://127.0.0.1:15060
```

可以通过 proxy backend server 地址进行访问： 

`http://0.0.0.0:8073/demo/demo`

::: tip
后端调试默认是最后一次操作后3秒启动后端编译启动命令。

hade 也允许通过配置修改这个等待时间。

可以配置 `development/app.yaml` 里面的 `dev_fresh` 参数修改这个等待时间。
:::

# 同时调试

也可以选择同时调试，这个时候会同时运行调试前端和调试后端的程序

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade dev all

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

```

前端和后端的访问地址分别为：

```
proxy backend server: http://0.0.0.0:8073
proxy frontend server: http://0.0.0.0:8073/dist/#/
```