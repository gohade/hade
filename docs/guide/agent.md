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

## 启动与演示

```
./hade agent start
```

需要本机 `database.default` 可连（与 `/demo/orm` 相同）。首次调用 `create_user` 会 AutoMigrate `users` 表。

```
curl -s -X POST http://127.0.0.1:8889/sessions
curl -N -X POST http://127.0.0.1:8889/sessions/<id>/messages \
  -H 'Content-Type: application/json' \
  -d '{"message":"创建一个名叫 foo、邮箱 foo@gmail.com、25 岁的用户，然后用返回的 id 再查一次。"}'
```

SSE 中应出现 `create_user` / `get_user` 的 `action` 与 JSON `observation`，并以 `final` 结束。
