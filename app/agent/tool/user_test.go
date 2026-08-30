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
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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
