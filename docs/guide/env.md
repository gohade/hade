# 环境变量

## 设置

hade 支持使用应用默认下的隐藏文件 `.env` 来配置各个机器不同的环境变量。

```
APP_ENV=development

DB_PASSWORD=mypassword
```

环境变量的设置可以在配置文件中通过 `env([环境变量])` 来获取到。

比如：

```
mysql:
    hostname: 127.0.0.1
    username: yejianfeng
    password:  env(DB_PASSWORD)
    timeout: 1
    readtime: 2.3
```


## 应用环境

hade 启动应用的默认应用环境为 development。

你可以通过设置 .env 文件中的 APP_ENV 设置应用环境。

应用环境建议选择：
- development // 开发使用
- production // 线上使用
- testing //测试环境

应用环境对应配置的文件夹，配置服务会去对应应用环境的文件夹中寻找配置。

比如应用环境为 development，在代码中使用
```
configService := container.MustMake(contract.ConfigKey).(contract.Config)
url := configService.GetString("app.url")
```

查找文件为：`config/development/app.yaml`

通过命令`./hade env`也可以获取当前应用环境：

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade env
environment: development
```