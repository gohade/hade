# 定时任务

## 指南

hade 中的定时任务是以命令的形式存在。hade 中也定义了一个命令 `./hade cron` 来对定时任务服务进行管理。

```
about cron command

Usage:
  hade cron [flags]
  hade cron [command]

Available Commands:
  list        list all cron command
  restart     restart cron command
  start       start cron command
  state       cron serve state
  stop        stop cron command

Flags:
  -h, --help   help for cron

Use "hade cron [command] --help" for more information about a command.
```

# 创建

创建一个定时任务和创建命令（command）是一致的。具体参考[command](/guide/command)

# 挂载

和挂载命令稍微有些不同，使用的方法是 `AddCronCommand`

```
rootCmd.AddCronCommand("* * * * *", command.DemoCommand)
```

# 查询

查询哪些定时任务挂载在服务上，使用命令 `./hade cron list`

```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade cron list
* * * * *  demo  demo
```

# 启动

使用命令 `./hade cron start` 启动一个定时服务
```
[~/Documents/workspace/hade_workspace/demo5]$ ./hade cron start
start cron job
[PID] 35453
```

也可以通过 `./hade cron start -d` 使用 deamon 模式启动一个定时服务

定时服务的输出记录在 `/storage/log/cron.log`

进程 id 记录在 `/storage/pid/app.pid`

# 状态

使用 deamon 模式启动定时服务的时候，可以使用命令 `./hade cron state` 查询定时任务状态

# 停止

使用 deamon 模式启动定时服务的时候，可以使用命令 `./hade cron stop` 停止定时任务

# 重启

使用 deamon 模式启动定时服务的时候，可以使用命令 `./hade cron restart` 重启定时任务


::: tip
如果程序还未启动，调用 restart 命令，效果和 start 命令一样，deamon 模式启动定时服务
:::


