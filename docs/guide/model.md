# 模型

## 指南

hade model 提供了自动生成数据库模型的命令行工具，如果你已经定义好你的model，这里的系列工具能帮助你节省不少时间。

```shell
> ./hade model
数据库模型相关的命令

Usage:
  hade model [flags]
  hade model [command]

Available Commands:
  api         通过数据库生成api
  gen         生成模型
  test        测试数据库

Flags:
  -h, --help   help for model
```

包含三个命令：

* ./hade model api 通过数据表自动生成api代码
* ./hade model gen 通过数据表自动生成model代码
* ./hade model test 测试某个数据库是否可以连接

## ./hade test

当你想测试下你的某个配置好的数据库是否能连接上，都有哪些表的时候，这个命令能帮助你。

```shell
> ./hade model test --database="database.default"
数据库连接：database.default成功
一共存在1张表
student
```

## ./hade model gen

这个命令帮助你生成数据表对应的gorm的model

```shell
 ✘  ~/Documents/workspace/gohade/hade   feature/model-gen ●  ./hade model gen
Error: required flag(s) "output" not set
Usage:
  hade model gen [flags]

Flags:
  -d, --database string   模型连接的数据库 (default "database.default")
  -h, --help              help for gen
  -o, --output string     模型输出地址

```

其中接受两个参数：

database: 这个参数可选，用来表示模型连接的数据库配置地址，默认是database.default，表示config目录下的{env}目录下的database.yaml中的default配置。

output：这个参数必填，用来表示模型文件的输出地址，如果填写相对路径，会在前面填充当前执行路径来补充为绝对路径。

### 使用方式

第一步，使用 `./hade model gen --output=app/model`

![image-20220215091522181](http://tuchuang.funaio.cn/img/image-20220215091522181.png)

选择其中的两个表，answers和questions，提示目录文件

![image-20220215091541908](http://tuchuang.funaio.cn/img/image-20220215091541908.png)

下一步确认y继续

![image-20220215091643242](http://tuchuang.funaio.cn/img/image-20220215091643242.png)

最后生成模型成功

![image-20220215091659104](http://tuchuang.funaio.cn/img/image-20220215091659104.png)

查看文件，确实生成了model

![image-20220215091735434](http://tuchuang.funaio.cn/img/image-20220215091735434.png)

## ./hade model api

```shell
> ./hade model api --database=database.default --output=/Users/jianfengye/Documents/workspace/gohade/hade/app/http/module/student/
```

![](http://tuchuang.funaio.cn/markdown/202303140005210.png)

它会在目标文件夹中生成对这个数据表的5个接口文件

* gen_[table]_api_create.go
* gen_[table]_api_delete.go
* gen_[table]_api_list.go
* gen_[table]_api_show.go
* gen_[table]_api_update.go

还有另外两个文件

* gen_[table]_model.go // 数据表对应的模型结构
* gen_[table]_router.go // 5个接口对应的默认路由

> 这里的每个文件都是可以修改的。
>
> 但是注意，如果重新生成，会覆盖原先的文件。
>
> 执行命令的时候切记保存已经改动的代码文件。
