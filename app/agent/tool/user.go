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
