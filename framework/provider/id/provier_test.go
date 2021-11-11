package id

import (
	tests "github.com/gohade/hade/test"
	"testing"

	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConsoleLog_Normal(t *testing.T) {
	Convey("test hade console log normal case", t, func() {
		c := tests.InitBaseContainer()
		c.Bind(&config.HadeConfigProvider{})

		err := c.Bind(&HadeIDProvider{})
		So(err, ShouldBeNil)

		idService := c.MustMake(contract.IDKey).(contract.IDService)
		xid := idService.NewID()
		So(xid, ShouldNotBeEmpty)
	})
}
