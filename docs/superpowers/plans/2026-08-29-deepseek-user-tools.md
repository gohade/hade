# DeepSeek User ORM 工具 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 DeepSeek 作为 LLM，在 Agent 进程内注册 `create_user` / `get_user` / `list_users`，直接对 `database.default` 上的 `demo.User` 做演示级 CRUD。

**Architecture:** 不改 ReAct / SSE / Kernel 协议。LLM 仍走 OpenAI 兼容 HTTP，只改 `llm.yaml` 指向 DeepSeek。User 工具在 `app/agent/tool/user.go` 用可注入的 `*gorm.DB` 函数实现；注册闭包从容器 `GetDB("database.default")`。无 ORM 绑定时不注册这三个工具。

**Tech Stack:** Go 1.25、已有 `hade:llm` / `hade:agent` / `hade:orm`、GORM、`gorm.io/driver/sqlite`（单测）、goconvey、DeepSeek Chat Completions（`deepseek-chat`）。

## Global Constraints

- 方案 A：Agent 进程内直接 ORM，复用 `demo.User` 与 `database.default`，不抽 repository。
- 不改 Kernel / SSE / `hade:llm` / `hade:agent` 协议与 ReAct 循环。
- 不做 `update_user` / `delete_user`；不改 `/demo/orm` 演示流。
- 工具所有可预期结果（成功、缺参、找不到、连不上库）返回 `error == nil` 且 observation 为 JSON（含 `ok`）。
- `AutoMigrate` 用 mutex + bool：成功才置位，失败可重试；不在 `NewAgentEngine` 构造期连库。
- 容器未绑定 `hade:orm` 时只注册 `echo` / `time`。
- 密钥用 `env(DEEPSEEK_API_KEY)`，不写入仓库；自动化测试不打真实 DeepSeek / 真实 MySQL。
- 提交信息使用中文；不要改无关文件。

## File structure

| Path | Responsibility |
|------|----------------|
| `config/development/llm.yaml` | DeepSeek `base_url` / `model` / `env(DEEPSEEK_API_KEY)` |
| `config/production/llm.yaml` | 同上，线上也走 DeepSeek |
| `config/testing/llm.yaml` | 保持空 key，测试不连真实模型 |
| `docs/guide/agent.md` | 如何配 DeepSeek、启动 agent、演示对话 |
| `app/agent/tool/user.go` | JSON 观察、migrate、CreateUser/GetUser/ListUsers、Handler 闭包 |
| `app/agent/tool/user_test.go` | sqlite 内存库单测 |
| `app/agent/kernel.go` | `RegisterExampleTools(agent, lookup)`，有 ORM 才注册 User 工具 |
| `app/agent/kernel_tools_test.go` | 无 ORM 不注册 User 工具 |

---

### Task 1: DeepSeek 配置与启动说明

**Files:**
- Modify: `config/development/llm.yaml`
- Modify: `config/production/llm.yaml`
- Create: `docs/guide/agent.md`

**Interfaces:**
- Consumes: 已有 `HadeLLMProvider` 读取 `llm.base_url` / `llm.api_key` / `llm.model`；Config `env()`。
- Produces: development/production 默认模型为 DeepSeek；文档说明 `.env` 与 `hade agent start`。

- [ ] **Step 1: 改 development 与 production 的 llm.yaml**

`config/development/llm.yaml` 与 `config/production/llm.yaml` 全文改为：

```yaml
base_url: "https://api.deepseek.com/v1"
api_key: env(DEEPSEEK_API_KEY)
model: "deepseek-chat"
```

不要改 `config/testing/llm.yaml`（继续空 `api_key`，避免测试误连外网）。

- [ ] **Step 2: 写 `docs/guide/agent.md`**

```markdown
# Agent

独立进程，默认端口 `:8889`（`config/{env}/agent.yaml` 的 `port`）。

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
```

注意：外层文档用 markdown 代码块时，内层 shell 围栏不要与外层冲突；实现时文件内容以上面为准。

- [ ] **Step 3: Commit**

```bash
git add config/development/llm.yaml config/production/llm.yaml docs/guide/agent.md
git commit -m "$(cat <<'EOF'
配置 DeepSeek 作为默认 LLM，并补充 Agent 启动说明。

EOF
)"
```

---

### Task 2: User ORM 工具函数（sqlite 单测）

**Files:**
- Create: `app/agent/tool/user.go`
- Create: `app/agent/tool/user_test.go`

**Interfaces:**
- Consumes: `demo.User`；`*gorm.DB`；`context.Context`。
- Produces:
  - `func CreateUser(ctx context.Context, db *gorm.DB, argsJSON string) (string, error)`
  - `func GetUser(ctx context.Context, db *gorm.DB, argsJSON string) (string, error)`
  - `func ListUsers(ctx context.Context, db *gorm.DB, argsJSON string) (string, error)`
  - 所有返回 `error == nil`；JSON 字段：`ok`、失败时 `error`；用户字段 `id`/`name`/`email`/`age`；列表为 `users` 数组，最多 20 条。

- [ ] **Step 1: 写失败测试 `app/agent/tool/user_test.go`**

```go
package tool

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/gohade/hade/app/http/module/demo"
	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&demo.User{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateUser_RequiresName(t *testing.T) {
	Convey("name 为空时 ok 为 false 且不写库", t, func() {
		db := openUserDB(t)
		out, err := CreateUser(context.Background(), db, `{}`)
		So(err, ShouldBeNil)
		So(out, ShouldContainSubstring, `"ok":false`)
		So(out, ShouldContainSubstring, "name is required")
		var n int64
		So(db.Model(&demo.User{}).Count(&n).Error, ShouldBeNil)
		So(n, ShouldEqual, 0)
	})
}

func TestCreateUser_ThenGetUser(t *testing.T) {
	Convey("创建后可按 id 查到", t, func() {
		db := openUserDB(t)
		out, err := CreateUser(context.Background(), db,
			`{"name":"foo","email":"foo@gmail.com","age":25}`)
		So(err, ShouldBeNil)
		var created struct {
			OK    bool   `json:"ok"`
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   uint8  `json:"age"`
		}
		So(json.Unmarshal([]byte(out), &created), ShouldBeNil)
		So(created.OK, ShouldBeTrue)
		So(created.ID, ShouldBeGreaterThan, 0)
		So(created.Name, ShouldEqual, "foo")
		So(created.Email, ShouldEqual, "foo@gmail.com")
		So(created.Age, ShouldEqual, 25)

		got, err := GetUser(context.Background(), db, `{"id":`+strconv.FormatUint(uint64(created.ID), 10)+`}`)
		So(err, ShouldBeNil)
		So(json.Unmarshal([]byte(got), &created), ShouldBeNil)
		So(created.OK, ShouldBeTrue)
		So(created.Name, ShouldEqual, "foo")
	})
}

func TestGetUser_NotFound(t *testing.T) {
	Convey("不存在的 id 返回 user not found", t, func() {
		db := openUserDB(t)
		out, err := GetUser(context.Background(), db, `{"id":999}`)
		So(err, ShouldBeNil)
		So(out, ShouldContainSubstring, `"ok":false`)
		So(out, ShouldContainSubstring, "user not found")
	})
}

func TestListUsers_FilterAndLimit(t *testing.T) {
	Convey("按 name 过滤，且最多 20 条", t, func() {
		db := openUserDB(t)
		ctx := context.Background()
		_, err := CreateUser(ctx, db, `{"name":"alpha"}`)
		So(err, ShouldBeNil)
		_, err = CreateUser(ctx, db, `{"name":"beta"}`)
		So(err, ShouldBeNil)
		listed, err := ListUsers(ctx, db, `{"name":"alp"}`)
		So(err, ShouldBeNil)
		var payload struct {
			OK    bool `json:"ok"`
			Users []struct {
				Name string `json:"name"`
			} `json:"users"`
		}
		So(json.Unmarshal([]byte(listed), &payload), ShouldBeNil)
		So(payload.OK, ShouldBeTrue)
		So(len(payload.Users), ShouldEqual, 1)
		So(payload.Users[0].Name, ShouldEqual, "alpha")

		for i := 0; i < 21; i++ {
			_, err = CreateUser(ctx, db, `{"name":"bulk"}`)
			So(err, ShouldBeNil)
		}
		all, err := ListUsers(ctx, db, `{"name":"bulk"}`)
		So(err, ShouldBeNil)
		So(json.Unmarshal([]byte(all), &payload), ShouldBeNil)
		So(len(payload.Users), ShouldEqual, 20)
	})
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./app/agent/tool -count=1 -run 'TestCreateUser|TestGetUser|TestListUsers'`

Expected: FAIL，未定义 `CreateUser` / `GetUser` / `ListUsers`。

- [ ] **Step 3: 实现 `app/agent/tool/user.go`**

```go
package tool

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/gohade/hade/app/http/module/demo"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/orm"
	"gorm.io/gorm"
)

// MustMaker 能从容器取出服务；*gin.Context 与 *framework.HadeContainer 均满足。
type MustMaker interface {
	MustMake(key string) interface{}
}

type userRow struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   uint8  `json:"age"`
}

func observationOK(row userRow) (string, error) {
	b, err := json.Marshal(struct {
		OK bool `json:"ok"`
		userRow
	}{OK: true, userRow: row})
	if err != nil {
		return `{"ok":false,"error":"encode failed"}`, nil
	}
	return string(b), nil
}

func observationFail(msg string) (string, error) {
	b, _ := json.Marshal(map[string]interface{}{"ok": false, "error": msg})
	return string(b), nil
}

func rowFromUser(u *demo.User) userRow {
	email := ""
	if u.Email != nil {
		email = *u.Email
	}
	return userRow{ID: u.ID, Name: u.Name, Email: email, Age: u.Age}
}

// CreateUser 插入一条 User。argsJSON：name 必填，email/age 可选。
func CreateUser(ctx context.Context, db *gorm.DB, argsJSON string) (string, error) {
	var args struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   uint8  `json:"age"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return observationFail("name is required")
	}
	user := &demo.User{Name: name, Age: args.Age}
	if strings.TrimSpace(args.Email) != "" {
		email := strings.TrimSpace(args.Email)
		user.Email = &email
	}
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		return observationFail("create user failed")
	}
	return observationOK(rowFromUser(user))
}

// GetUser 按 id 查询。
func GetUser(ctx context.Context, db *gorm.DB, argsJSON string) (string, error) {
	var args struct {
		ID uint `json:"id"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args.ID == 0 {
		return observationFail("id is required")
	}
	user := &demo.User{}
	err := db.WithContext(ctx).First(user, args.ID).Error
	if err == gorm.ErrRecordNotFound {
		return observationFail("user not found")
	}
	if err != nil {
		return observationFail("query user failed")
	}
	return observationOK(rowFromUser(user))
}

// ListUsers 最多 20 条；name 非空时 LIKE 过滤。
func ListUsers(ctx context.Context, db *gorm.DB, argsJSON string) (string, error) {
	var args struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	q := db.WithContext(ctx).Model(&demo.User{}).Limit(20)
	name := strings.TrimSpace(args.Name)
	if name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	var users []demo.User
	if err := q.Find(&users).Error; err != nil {
		return observationFail("list users failed")
	}
	rows := make([]userRow, 0, len(users))
	for i := range users {
		rows = append(rows, rowFromUser(&users[i]))
	}
	b, err := json.Marshal(map[string]interface{}{"ok": true, "users": rows})
	if err != nil {
		return observationFail("encode failed")
	}
	return string(b), nil
}

var (
	migrateMu   sync.Mutex
	migrateDone bool
)

func ensureUserTable(db *gorm.DB) string {
	migrateMu.Lock()
	defer migrateMu.Unlock()
	if migrateDone {
		return ""
	}
	if err := db.AutoMigrate(&demo.User{}); err != nil {
		return "migrate users failed"
	}
	migrateDone = true
	return ""
}

func openDefaultDB(m MustMaker, ctx context.Context) (*gorm.DB, string) {
	ormService, ok := m.MustMake(contract.ORMKey).(contract.ORMService)
	if !ok {
		return nil, "database unavailable"
	}
	db, err := ormService.GetDB(orm.WithConfigPath("database.default"))
	if err != nil || db == nil {
		return nil, "database unavailable"
	}
	return db.WithContext(ctx), ""
}

func withDefaultUserDB(m MustMaker, ctx context.Context) (*gorm.DB, string) {
	db, fail := openDefaultDB(m, ctx)
	if fail != "" {
		return nil, fail
	}
	if msg := ensureUserTable(db); msg != "" {
		return nil, msg
	}
	return db, ""
}

// CreateUserHandler 从容器取 database.default 后调用 CreateUser。
func CreateUserHandler(m MustMaker) contract.ToolHandler {
	return func(ctx context.Context, argsJSON string) (string, error) {
		db, fail := withDefaultUserDB(m, ctx)
		if fail != "" {
			return observationFail(fail)
		}
		return CreateUser(ctx, db, argsJSON)
	}
}

// GetUserHandler 从容器取 database.default 后调用 GetUser。
func GetUserHandler(m MustMaker) contract.ToolHandler {
	return func(ctx context.Context, argsJSON string) (string, error) {
		db, fail := withDefaultUserDB(m, ctx)
		if fail != "" {
			return observationFail(fail)
		}
		return GetUser(ctx, db, argsJSON)
	}
}

// ListUsersHandler 从容器取 database.default 后调用 ListUsers。
func ListUsersHandler(m MustMaker) contract.ToolHandler {
	return func(ctx context.Context, argsJSON string) (string, error) {
		db, fail := withDefaultUserDB(m, ctx)
		if fail != "" {
			return observationFail(fail)
		}
		return ListUsers(ctx, db, argsJSON)
	}
}
```

Task 2 的 sqlite 测试只调用 `CreateUser`/`GetUser`/`ListUsers`，不经过 Handler。Handler 在本任务一并写好，供 Task 3 注册。

- [ ] **Step 4: 再跑测试，确认通过**

Run: `go test ./app/agent/tool -count=1 -run 'TestCreateUser|TestGetUser|TestListUsers'`

Expected: PASS。sqlite 若因 CGO 失败，把 Open 改成同一进程内已工作的 DSN，不要改成跳过测试。

- [ ] **Step 5: Commit**

```bash
git add app/agent/tool/user.go app/agent/tool/user_test.go
git commit -m "$(cat <<'EOF'
增加可演示的 User 创建与查询 Agent 工具。

EOF
)"
```

---

### Task 3: 注册 User 工具（无 ORM 则跳过）

**Files:**
- Modify: `app/agent/kernel.go`（`registerExampleToolsSafely`、`RegisterExampleTools`、`resolve` 调用）
- Create: `app/agent/kernel_tools_test.go`

**Interfaces:**
- Consumes: Task 2 的 `tool.MustMaker`、`CreateUserHandler` / `GetUserHandler` / `ListUsersHandler`；`contract.ORMKey`；`echo`/`time` 现有注册。
- Produces: `RegisterExampleTools(agent contract.Agent, lookup serviceLookup)`。`serviceLookup` 为 `IsBind` + `MustMake`。`lookup == nil` 或 `!IsBind(ORMKey)` 时不注册三个 User 工具。

`*framework.HadeContainer` 与 `*gin.Context` 都满足 `serviceLookup`。Task 2 的 `tool.MustMaker` 只含 `MustMake`，因此 `CreateUserHandler(lookup)` 合法。

- [ ] **Step 1: 写 `app/agent/kernel_tools_test.go`**

```go
package agent

import (
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	agprovider "github.com/gohade/hade/framework/provider/agent"
	llmp "github.com/gohade/hade/framework/provider/llm"
	. "github.com/smartystreets/goconvey/convey"
)

func toolNames(agent contract.Agent) map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range agent.ListTools() {
		out[spec.Name] = struct{}{}
	}
	return out
}

func TestRegisterExampleTools_SkipsUserToolsWithoutORM(t *testing.T) {
	Convey("未绑定 ORM 时只有 echo 和 time", t, func() {
		container := framework.NewHadeContainer()
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "echo")
		So(names, ShouldContainKey, "time")
		So(names, ShouldNotContainKey, "create_user")
		So(names, ShouldNotContainKey, "get_user")
		So(names, ShouldNotContainKey, "list_users")
	})
}

type stubORMProvider struct{}

func (p *stubORMProvider) Name() string { return contract.ORMKey }
func (p *stubORMProvider) Register(framework.Container) framework.NewInstance {
	return func(params ...interface{}) (interface{}, error) { return struct{}{}, nil }
}
func (p *stubORMProvider) Boot(framework.Container) error           { return nil }
func (p *stubORMProvider) IsDefer() bool                            { return true }
func (p *stubORMProvider) Params(framework.Container) []interface{} { return nil }

func TestRegisterExampleTools_RegistersUserToolsWhenORMBound(t *testing.T) {
	Convey("绑定 ORM 关键字后注册三个 User 工具", t, func() {
		container := framework.NewHadeContainer()
		So(container.Bind(&stubORMProvider{}), ShouldBeNil)
		mem := agprovider.NewMemoryAgent(&llmp.ScriptLLM{}, 8)
		RegisterExampleTools(mem, container)
		names := toolNames(mem)
		So(names, ShouldContainKey, "create_user")
		So(names, ShouldContainKey, "get_user")
		So(names, ShouldContainKey, "list_users")
	})
}
```

本测试只断言注册了工具名，不调用 Handler。

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./app/agent -count=1 -run TestRegisterExampleTools`

Expected: FAIL（`RegisterExampleTools` 仍是单参数，或不注册 User 工具）。

- [ ] **Step 3: 改 kernel.go**

1. 增加 `serviceLookup` 接口（`IsBind` + `MustMake`）。
2. `registerExampleToolsSafely(agent, lookup)` 调用 `RegisterExampleTools(agent, lookup)`。
3. `resolve` 里：`registerExampleToolsSafely(r.agent, c)`。
4. `RegisterExampleTools` 全文如下：

```go
func RegisterExampleTools(agent contract.Agent, lookup serviceLookup) {
	if agent == nil {
		return
	}
	registered := map[string]struct{}{}
	for _, spec := range agent.ListTools() {
		registered[spec.Name] = struct{}{}
	}
	if _, ok := registered["echo"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "echo",
			Description: "返回输入文本",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{"type": "string"},
				},
			},
		}, tool.Echo)
	}
	if _, ok := registered["time"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "time",
			Description: "返回当前 UTC 时间",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}, tool.Time)
	}
	if lookup == nil || !lookup.IsBind(contract.ORMKey) {
		return
	}
	if _, ok := registered["create_user"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "create_user",
			Description: "在数据库中创建用户，返回新记录的 id 与字段",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]interface{}{
					"name":  map[string]interface{}{"type": "string"},
					"email": map[string]interface{}{"type": "string"},
					"age":   map[string]interface{}{"type": "integer"},
				},
			},
		}, tool.CreateUserHandler(lookup))
	}
	if _, ok := registered["get_user"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "get_user",
			Description: "按主键 id 查询一个用户",
			Parameters: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer"},
				},
			},
		}, tool.GetUserHandler(lookup))
	}
	if _, ok := registered["list_users"]; !ok {
		agent.RegisterTool(contract.ToolSpec{
			Name:        "list_users",
			Description: "列出用户，最多 20 条；可选按 name 模糊匹配",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
		}, tool.ListUsersHandler(lookup))
	}
}
```

`serviceLookup`：

```go
type serviceLookup interface {
	IsBind(key string) bool
	MustMake(key string) interface{}
}
```

- [ ] **Step 4: 跑注册测试与现有 agent 包测试**

Run:

```
go test ./app/agent -count=1 -run TestRegisterExampleTools
go test ./app/agent ./app/agent/tool -count=1
```

Expected: 全部 PASS。`engine_*` 测试容器通常无 ORM，resolve 时只注册 echo/time，行为与现在一致。

- [ ] **Step 5: Commit**

```bash
git add app/agent/kernel.go app/agent/kernel_tools_test.go app/agent/tool/user.go
git commit -m "$(cat <<'EOF'
在 Agent 引擎注册 User ORM 工具，无 ORM 时跳过。

EOF
)"
```

---

## 手动验收（不写入自动化）

本机 MySQL 按 `config/development/database.yaml` 可连，`.env` 有 `DEEPSEEK_API_KEY`：

```
go run . agent start
```

按 `docs/guide/agent.md` 发创建+查询。不把 key 或 curl 输出提交进仓库。

---

## Spec coverage

| Spec 项 | Task |
|---------|------|
| DeepSeek llm.yaml + env key | 1 |
| 文档一句/启动说明 | 1 |
| create/get/list + JSON ok | 2 |
| sqlite 单测、不打 DeepSeek/MySQL | 2 |
| 复用 demo.User、database.default | 2 handlers + 3 注册 |
| AutoMigrate mutex+bool、非构造期 | 2 `ensureUserTable` |
| 无 ORM 不注册 User 工具 | 3 |
| 不抽 repository、不改 ReAct/SSE | 全局，无对应协议改动任务 |
| testing llm 空 key | 1 明确不改 testing yaml |
