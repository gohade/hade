# DeepSeek ReAct 演示：User ORM 工具

日期：2026-08-29  
状态：待用户确认后进入实现计划

## 背景与目标

hade Agent Engine 已提供服务端 ReAct + SSE，LLM 层为 OpenAI 兼容 Chat Completions。本设计在 **不改 Kernel / SSE / ReAct 循环** 的前提下，接 DeepSeek，并增加 2～3 个可演示的业务工具。

工具对齐 `DemoOrm`（`app/http/module/demo/api_orm.go`）：把 `User` 写入 `database.default`，再按 ID / 列表查出来。工具在 **Agent 进程内直接调 ORM**（方案 A），与 HTTP `/demo/orm` 写同一张表。

**成功标准：**

- `hade agent start` 后，用自然语言让 DeepSeek 创建用户并查询，SSE 中出现 `create_user` / `get_user`（及可选 `list_users`）的 `action` / `observation`，并以 `final` 结束。
- 单测不调用真实 DeepSeek、不依赖本机 MySQL；工具 handler 用 sqlite 内存库验证。

## 明确不做

- 不抽 User repository，不改 `/demo/orm` 的增改查删演示流。
- 不提供 `update_user` / `delete_user`。
- 不把 ORM 提升为框架内置 Agent 工具；不改 `hade:llm` / `hade:agent` 协议。
- 自动化测试不打真实 DeepSeek 网络。
- 不把 API Key 写入仓库或 yaml 明文。

## 架构

```
DeepSeek Chat Completions
        ↑ OpenAI 兼容 HTTP（已有 HadeLLMProvider）
hade:llm  ← config/development/llm.yaml
hade:agent ReAct
        ↓ RegisterTool（进程内）
create_user / get_user / list_users
        ↓ contract.ORMService.GetDB(orm.WithConfigPath("database.default"))
demo.User 表（与 DemoOrm 相同）
```

启动路径不变：`main` 已绑定 `GormProvider` 与 LLM/Agent；`hade agent start` 拉起 Agent 端口（默认 `:8889`）。

### 组件

| 单元 | 路径 | 职责 | 依赖 |
|------|------|------|------|
| DeepSeek 配置 | `config/development/llm.yaml` + `.env` | `base_url` / `model` / `api_key` | 已有 Config `env()` |
| 工具注册 | `app/agent/kernel.go` `RegisterExampleTools` | 首次 resolve Agent 时注册 echo/time/User 工具 | Container、Agent |
| User 工具 | `app/agent/tool/user.go` | 解析 JSON 参数、AutoMigrate（一次）、CRUD 观察 | `demo.User`、`*gorm.DB` |
| 取库 | 注册闭包从 Container `MustMake(ORMKey)` | 与 DemoOrm 相同的 `database.default` | `hade:orm` |

`app/agent` 可以 import `app/http/module/demo` 以复用 `User`：`demo` 不引用 `app/agent`，无循环依赖。不把模型再抽到新包。

## 配置

`config/development/llm.yaml`：

```yaml
base_url: "https://api.deepseek.com/v1"
api_key: env(DEEPSEEK_API_KEY)
model: "deepseek-chat"
```

`production` / `testing` 的 `llm.yaml` 同步字段含义，testing 可继续空 key（测试走 fake LLM）。密钥只放 `.env` 的 `DEEPSEEK_API_KEY`。模型固定 `deepseek-chat`（需要 function calling）。数据库仍用现有 `config/{env}/database.yaml` 的 `default`，不新增 agent 专用库配置。

## 工具契约

Handler 签名不变：`func(ctx context.Context, argsJSON string) (observation string, error)`。  
**所有可预期结果（成功、缺参、找不到行、连不上库）都返回 `error == nil`**，`observation` 为 JSON（含 `ok`）。ReAct 不因这些情况中断。JSON 观察仍受现有 `ContentMaxBytes`（4096）截断。

### `create_user`

- Description（给模型）：在数据库中创建用户，返回新记录的 id 与字段。
- Parameters：`name` string 必填；`email` string 可选；`age` integer 可选（0–255，缺省 0）。
- 行为：确保 `users` 表已 migrate（见下）；`db.Create`；observation 含 `ok`、`id`、`name`、`email`、`age`。
- `name` 为空：`ok: false`，`error: "name is required"`。

### `get_user`

- Description：按主键 id 查询一个用户。
- Parameters：`id` integer 必填。
- 找到：`ok: true` 及字段；找不到：`ok: false`，`error: "user not found"`（不把 gorm 内部错误原文全量抛给模型）。

### `list_users`

- Description：列出用户，最多 20 条；可选按 name 模糊匹配。
- Parameters：`name` string 可选。
- 行为：`Limit(20)`；若 `name` 非空则 `Name LIKE %name%`；observation 为 `ok: true` 与 `users` 数组（id/name/email/age）。

保留现有 `echo`、`time`。同名已注册则跳过（现有逻辑）。

## 取 DB 与 AutoMigrate

注册 User 工具时，handler 闭包捕获 `framework.Container`（`RegisterExampleTools(agent, container)`）。每次调用：

1. `ormService := container.MustMake(contract.ORMKey).(contract.ORMService)`
2. `db, err := ormService.GetDB(orm.WithConfigPath("database.default"))`
3. `db = db.WithContext(ctx)`

`AutoMigrate(&demo.User{})` **每个进程成功一次即可**。用 mutex + bool（与 `agentResolver.toolsReady` 相同）：成功才置位；失败则本次 observation `ok: false`，下次工具调用再试。不使用 `sync.Once`（失败也会把 Once 锁死）。

不在 `NewAgentEngine` 构造期连库或 migrate（保持「非 agent 命令不碰 LLM/ORM 副作用」；migrate 发生在首次 User 工具调用）。

## 数据流（演示）

1. 配置 DeepSeek 与 `.env`。
2. `hade agent start`。
3. `POST /sessions` → `POST /sessions/:id/messages`，message 例如：「创建一个名叫 foo、邮箱 foo@gmail.com、25 岁的用户，然后用返回的 id 再查一次。」
4. ReAct：thought → action `create_user` → observation（含 id）→ action `get_user` → observation → final。

## 错误与安全

- API Key 空：已有 LLM 层返回 `ErrLLMFailed`，不改。
- 数据库不可达：observation JSON `ok: false`，文案说明无法连接，不泄露密码。
- 不执行任意 SQL；参数只映射到结构化字段。
- `.env` 已在 gitignore 则不改；确认 `DEEPSEEK_API_KEY` 不会被提交。

## 测试

- `app/agent/tool/user_test.go`：sqlite 内存库打开 `*gorm.DB`，直接调用可注入 DB 的 create/get/list 函数；注册闭包只负责 `GetDB` 再转调这些函数。覆盖：创建后可查、缺 name、get 不存在、list 过滤与 20 条上限。
- `RegisterExampleTools(agent, container)`：`container` 未绑定 `hade:orm` 时只注册 `echo` / `time`，不注册 User 三工具（现有 engine 测试缺 ORM 时不 panic）。绑定了 ORM 则三个 User 工具都注册。
- 不新增真实 DeepSeek / 真实 MySQL 的 CI 测试。

## 实现顺序建议

1. llm.yaml + `.env` 示例说明（文档一句即可，不提交真实 key）。
2. `RegisterExampleTools(agent, container)` + User 工具与 sqlite 单测。
3. 手动：DeepSeek + 本机 `database.default` 跑通上述对话。
