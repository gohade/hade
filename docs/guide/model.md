# 模型

## 指南

hade 提供了自动生成数据库模型的命令行工具

```
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

## 使用方式

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


