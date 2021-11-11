package config

import (
	"testing"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/app"
	"github.com/gohade/hade/framework/provider/env"
	tests "github.com/gohade/hade/test"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHadeConfig_Normal(t *testing.T) {
	Convey("test hade config normal case", t, func() {
		basePath := tests.BasePath
		c := framework.NewHadeContainer()
		c.Bind(&app.HadeAppProvider{BaseFolder: basePath})
		c.Bind(&env.HadeEnvProvider{})

		err := c.Bind(&HadeConfigProvider{})
		So(err, ShouldBeNil)

		conf := c.MustMake(contract.ConfigKey).(contract.Config)
		So(conf.GetString("database.default.host"), ShouldEqual, "localhost")
		So(conf.GetInt("database.default.port"), ShouldEqual, 3306)
		//So(conf.GetFloat64("database.default.readtime"), ShouldEqual, 2.3)
		// So(conf.GetString("database.mysql.password"), ShouldEqual, "mypassword")

		maps := conf.GetStringMap("database.default")
		So(maps, ShouldContainKey, "host")
		So(maps["host"], ShouldEqual, "localhost")

		maps2 := conf.GetStringMapString("database.default")
		So(maps2["host"], ShouldEqual, "localhost")

		type Mysql struct {
			Host string `yaml:"host"`
		}
		ms := &Mysql{}
		err = conf.Load("database.default", ms)
		So(err, ShouldBeNil)
		So(ms.Host, ShouldEqual, "localhost")
	})
}
