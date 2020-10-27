package env

import (
	"testing"

	"hade/framework"
	"hade/framework/contract"
	"hade/framework/provider/app"
	"hade/tests"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHadeEnvProvider(t *testing.T) {
	Convey("test hade env normal case", t, func() {
		basePath := tests.BasePath
		c := framework.NewHadeContainer()
		sp := &app.HadeAppProvider{BasePath: basePath}

		err := c.Singleton(sp)
		So(err, ShouldBeNil)

		sp2 := &HadeEnvProvider{}
		err = c.Singleton(sp2)
		So(err, ShouldBeNil)

		envServ := c.MustMake(contract.EnvKey).(contract.Env)
		So(envServ.AppEnv(), ShouldEqual, "development")
		// So(envServ.Get("DB_HOST"), ShouldEqual, "127.0.0.1")
		// So(envServ.AppDebug(), ShouldBeTrue)
	})
}
