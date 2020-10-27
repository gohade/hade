# 编译

---

## 命令

应用分为前端（frontend）和后端（backend），所以编译也分为三类
- 编译前端 
- 编译后端 
- 自编译
- 同时编译

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade build
build hade command

Usage:
  hade build [flags]
  hade build [command]

Available Commands:
  all         build fronend and backend
  backend     build backend use go
  frontend    build frontend use npm
  self        build ./hade command

Flags:
  -h, --help   help for build

Use "hade build [command] --help" for more information about a command.
```

## 编译前端

要求当前编译机器安装 npm 软件，并且当前项目已经运行了 npm install，安装完成前端依赖。

运行命令 `./hade build frontend`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade build frontend

> hade@0.1.0 build /Users/Documents/workspace/hade_workspace/demo5
> vue-cli-service build


-  Building for production...
 DONE  Compiled successfully in 5012ms下午5:44:47

  File                                      Size             Gzipped

  dist/asset/js/chunk-vendors.222f9fef.j    117.96 KiB       42.76 KiB
  s
  dist/asset/js/index.0ed60f05.js           4.63 KiB         1.66 KiB
  dist/asset/css/index.fa7f2f34.css         0.33 KiB         0.23 KiB

  Images and other types of assets omitted.

 DONE  Build complete. The dist directory is ready to be deployed.
 INFO  Check out deployment instructions at https://cli.vuejs.org/guide/deployment.html

front end build success
```

编译后的前端文件在 dist 目录中

实际上 build 就是调用 `npm build` 来编译前端项目。


## 编译后端

要求当前编译机器安装 go 软件，版本 > 1.3。

运行命令： `./hade build backend`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade build backend
build success please run ./hade direct
```

在项目根目录下就看到生成的可执行文件 hade。 后续可以通过 ./hade 直接运行。

## 自编译

在项目根目录下，hade 可以通过 hade 命令编译出 hade 命令自己。通过命令 `hade build self`

```
[~/Documents/workspace/hade_workspace/demo5]$ hade build self
build success please run ./hade direct
```

在项目根目录下就看到生成的可执行文件 hade。 后续可以通过 ./hade 直接运行。

::: tip
其实自编译和后端编译是同样效果，但是为了命令语义化，增加了自编译的命令。
:::

## 同时编译

顾名思义，同时编译前端和后端，命令为 `./hade build all`