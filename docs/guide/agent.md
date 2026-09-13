# Agent

独立进程，默认端口 `:8889`（`config/{env}/agent.yaml` 的 `port`）。

Session 默认存在**进程内存**（`session_store: memory`）。`main.go` 即使绑定了 Redis Provider，也不会自动用 Redis 存对话。多实例共享时在 `agent.yaml` 设 `session_store: redis`，并保证 Redis 可连；Ping 失败会打 Warn 并回退内存，避免 `POST /sessions` 变成 `{"error":"internal"}`。

## DeepSeek

在项目根 `.env` 中设置（该文件已 gitignore）：

```
APP_ENV=development
DEEPSEEK_API_KEY=你的key
```

`config/development/llm.yaml` 使用 OpenAI 兼容接口：`https://api.deepseek.com/v1`，模型 `deepseek-chat`。

每次 `Chat` 会把请求体和响应体打到 **Debug** 日志（`log.level` 为 `debug`/`trace` 时可见），不记录 `Authorization` 和 API Key。development 默认 `log.level: trace`，rotate 文件在应用 log 目录。

## 启动与演示

```
./hade agent start
```

启动后访问 `http://127.0.0.1:8889/` 可打开内置的 Agent Console。页面不需要单独安装或构建前端依赖，已经随 Go 二进制一起编译。

`config/{env}/agent.yaml` 中的 `console_enabled` 控制调试台路由，默认开启：

```yaml
console_enabled: true
```

设置为 `false` 后，`/`、`/agent-ui.css`、`/agent-ui.js`、`GET/PUT /config` 和 `/tools` 均不再注册；Session 与 Messages 业务 API 不受影响。

Console 支持：

- 在运行时切换 OpenAI 兼容的 API Base URL、模型和 API Token；
- 新建、恢复当前浏览器标签页中的 Session，并进行多轮对话；
- 实时展示 `thought`、`action`、`observation`、`final` 和 `error` SSE 事件；
- 查看 Agent 当前注册的工具，以及中止正在执行的请求。

页面提交的模型配置仅影响**当前 Agent 进程**。Token 只保存在服务端内存中，查询接口不会回传原文，浏览器也不会把它写入 Web Storage。进程重启后仍以 `config/{env}/llm.yaml` 和环境变量为准。

调试接口当前不带身份认证。包含 Token 的场景建议用 `--address 127.0.0.1:8889` 只监听本机；如果需要开放到局域网或公网，应先在上游增加 HTTPS 与访问控制。

对应的调试接口为：

```
GET  /config     # 返回 base_url、model、api_key_configured
PUT  /config     # 热更新；省略 api_key 保留旧值，传空字符串清空
GET  /tools      # 返回当前注册的工具定义
```

需要本机 `database.default` 可连（与 `/demo/orm` 相同）。首次调用 `create_user` 会 AutoMigrate `users` 表。

```
curl -s -X POST http://127.0.0.1:8889/sessions
curl -N -X POST http://127.0.0.1:8889/sessions/<id>/messages \
  -H 'Content-Type: application/json' \
  -d '{"message":"创建一个名叫 foo、邮箱 foo@gmail.com、25 岁的用户，然后用返回的 id 再查一次。"}'
```

SSE 中应出现 `create_user` / `get_user` 的 `action` 与 JSON `observation`，并以 `final` 结束。
