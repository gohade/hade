package id

import (
	"testing"

	"hade/framework"
	"hade/framework/contract"
	"hade/framework/provider/app"
	"hade/framework/provider/config"
	"hade/framework/provider/env"
	"hade/framework/util"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConsoleLog_Normal(t *testing.T) {
	Convey("test hade console log normal case", t, func() {
		basePath := util.GetExecDirectory()
		c := framework.NewHadeContainer()
		c.Singleton(&app.HadeAppProvider{BasePath: basePath})
		c.Singleton(&env.HadeEnvProvider{})
		c.Singleton(&config.HadeConfigProvider{})

		err := c.Singleton(&HadeIDProvider{})
		So(err, ShouldBeNil)

		idService := c.MustMake(contract.IDKey).(contract.IDService)
		xid := idService.NewID()
		t.Log(xid)
		So(xid, ShouldNotBeEmpty)
	})
}
