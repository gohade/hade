# 版本

hade 提供了查询当前版本和获取最新版本日志的命令

## 查询当前版本

使用命令 `hade version`

```
➜ hade git:(main) ✗ ./hade version
hade version: 1.0.3
```

## 获取最新的版本

使用命令 `hade version list`

```
 ➜  hade git:(main) ✗ ./hade version list
===============前置条件检测===============
hade源码从github.com中下载，正在检测到github.com的连接
github.Rate{Limit:60, Remaining:29, Reset:github.Timestamp{2022-12-11 17:43:48 +0800 CST}}
hade源码从github.com中下载，github.com的连接正常
===============前置条件检测结束===============

最新的6个版本
-v1.0.2
  发布时间：2022-12-11 08:57:37
  修改说明：
    创建1.0.2
-v1.0.1
  发布时间：2021-12-28 08:22:58
  修改说明：
    1  增加了一个计划版本
-v1.0.0
  发布时间：2021-11-21 18:02:00
  修改说明：
    create release
-v0.0.3
  发布时间：2021-10-08 23:13:52
  修改说明：
    完善new命令
-v0.0.2
  发布时间：2021-10-06 09:19:44
  修改说明：
    1 增加了环境变量设置
    2 增加了前后端一体化
-v0.0.1
  发布时间：2021-10-06 09:20:28
  修改说明：
    第一版本

更多历史版本请参考 https://github.com/gohade/hade/releases
```
