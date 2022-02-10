package services

import (
	"context"
	"encoding/json"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/provider/config"
	"github.com/gohade/hade/framework/provider/log"
	tests "github.com/gohade/hade/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

type Bar struct {
	Name string
}

func (b *Bar) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Bar) UnmarshalBinary(bt []byte) error {
	return json.Unmarshal(bt, b)
}

func TestHadeRedisService_Load(t *testing.T) {
	container := tests.InitBaseContainer()
	container.Bind(&config.HadeConfigProvider{})
	container.Bind(&log.HadeLogServiceProvider{})

	Convey("test get client", t, func() {
		it, err := NewRedisCache(container)
		So(err, ShouldBeNil)
		mc, ok := it.(*RedisCache)
		So(ok, ShouldBeTrue)
		So(mc, ShouldNotBeNil)
		ctx := context.Background()

		Convey("string get set", func() {
			err := mc.Set(ctx, "foo", "bar", 1*time.Hour)
			So(err, ShouldBeNil)
			val, err := mc.Get(ctx, "foo")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, "bar")
			err = mc.SetTTL(ctx, "foo", 1*time.Minute)
			So(err, ShouldBeNil)
			du, err := mc.GetTTL(ctx, "foo")
			So(err, ShouldBeNil)
			So(du, ShouldBeLessThanOrEqualTo, 1*time.Minute)
			err = mc.Del(ctx, "foo")
			So(err, ShouldBeNil)
			val, err = mc.Get(ctx, "foo")
			So(err, ShouldEqual, ErrKeyNotFound)
		})

		Convey("obj get set", func() {
			obj := &Bar{
				Name: "bar",
			}
			err := mc.SetObj(ctx, "foo", obj, 1*time.Hour)
			So(err, ShouldBeNil)
			objNew := Bar{}
			err = mc.GetObj(ctx, "foo", &objNew)
			So(err, ShouldBeNil)
			So(objNew.Name, ShouldEqual, "bar")
			err = mc.Del(ctx, "foo")
			So(err, ShouldBeNil)
		})

		Convey("many op", func() {
			err = mc.SetMany(ctx, map[string]string{
				"foo1": "bar1",
				"foo2": "bar2",
			}, 1*time.Hour)
			So(err, ShouldBeNil)

			ret, err := mc.GetMany(ctx, []string{"foo1", "foo2"})
			So(err, ShouldBeNil)
			So(len(ret), ShouldEqual, 2)
			So(ret, ShouldContainKey, "foo2")
			So(ret["foo2"], ShouldEqual, "bar2")

			err = mc.DelMany(ctx, []string{"foo1", "foo2"})
			So(err, ShouldBeNil)
		})

		Convey("calc op", func() {
			val, err := mc.Increment(ctx, "foo")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, 1)
			val, err = mc.Calc(ctx, "foo", 2)
			So(err, ShouldBeNil)
			So(val, ShouldEqual, 3)
			val, err = mc.Decrement(ctx, "foo")
			So(err, ShouldBeNil)
			So(val, ShouldEqual, 2)
			err = mc.Del(ctx, "foo")
			So(err, ShouldBeNil)
		})

		Convey("remember op", func() {
			objNew := Bar{}
			objNewFunc := func(ctx context.Context, container framework.Container) (interface{}, error) {
				obj := &Bar{
					Name: "bar",
				}
				return obj, nil
			}
			err = mc.Remember(ctx, "foo_remember", 1*time.Minute, objNewFunc, &objNew)
			So(err, ShouldBeNil)
			So(objNew.Name, ShouldEqual, "bar")
		})
	})
}
