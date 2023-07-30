package sls

import (
    "github.com/gohade/hade/framework/provider/config"
    tests "github.com/gohade/hade/test"
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)

func TestHadeSLSService_Load(t *testing.T) {
    container := tests.InitBaseContainer()
    container.Bind(&config.HadeConfigProvider{})
    Convey("test", t, func() {
        slsService, err := NewHadeSLSService(container)
        So(err, ShouldBeNil)
        So(slsService, ShouldNotBeNil)
        service, ok := slsService.(*HadeSLSService)
        So(ok, ShouldBeTrue)
        _, err = service.GetSLSInstance()
        So(err, ShouldBeNil)
        project, err := service.GetProject()
        So(err, ShouldBeNil)
        So(project, ShouldNotBeNil)
        logstore, err := service.GetLogstore()
        So(err, ShouldBeNil)
        So(logstore, ShouldNotBeNil)
        service.producerInstance.SafeClose()
    })
}
