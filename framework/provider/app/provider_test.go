package app

import (
	"testing"

	"hade/framework"
	"hade/framework/contract"
	"hade/tests"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHadeAppProvider(t *testing.T) {
	Convey("test normal case", t, func() {
		basePath := tests.BasePath
		c := framework.NewHadeContainer()
		sp := &HadeAppProvider{BasePath: basePath}

		err := c.Singleton(sp)
		So(err, ShouldBeNil)

		app, err := c.Make(contract.AppKey)
		So(err, ShouldBeNil)
		var iapp *contract.App
		So(app, ShouldImplement, iapp)
		hade := app.(contract.App)
		logPath := hade.LogPath()
		So(logPath, ShouldEqual, basePath+"storage/log/")
	})
}
