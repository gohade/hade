package log

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hade/framework"
	"hade/framework/contract"
	"hade/framework/provider/app"
	"hade/framework/provider/config"
	"hade/framework/provider/env"
	"hade/framework/provider/id"
	"hade/framework/provider/log/formatter"
	"hade/framework/provider/trace"
	"hade/tests"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

func TestConsoleLog_Normal(t *testing.T) {
	Convey("test hade console log normal case", t, func() {
		basePath := tests.BasePath
		c := framework.NewHadeContainer()
		c.Singleton(&app.HadeAppProvider{BasePath: basePath})
		c.Singleton(&env.HadeEnvProvider{})
		c.Singleton(&config.HadeConfigProvider{})

		err := c.Singleton(&HadeLogServiceProvider{})
		So(err, ShouldBeNil)

		var buf bytes.Buffer
		l := c.MustMake(contract.LogKey).(contract.ConsoleLog)
		l.SetOutput(&buf)
		l.Debug(context.Background(), "test_debug", map[string]interface{}{"foo1": []int{1, 2, 3}})
		l.Info(context.Background(), "test_debug", map[string]interface{}{"foo2": []int{1, 2, 3}})
		So(buf.String(), ShouldNotContainSubstring, "foo1")
		So(buf.String(), ShouldContainSubstring, "foo2")

		buf.Reset()
		l.SetLevel(contract.InfoLevel)
		l.Debug(context.Background(), "test_debug", map[string]interface{}{"foo1": []int{1, 2, 3}})
		l.Info(context.Background(), "test_debug", map[string]interface{}{"foo2": []int{1, 2, 3}})
		So(buf.String(), ShouldNotContainSubstring, "foo1")

		buf.Reset()
		l.SetLevel(contract.InfoLevel)
		l.SetFormatter(formatter.JsonFormatter)
		ck := "foo"
		cv := "bar"
		ctx := context.WithValue(context.Background(), ck, cv)
		l.SetCxtFielder(func(ctx context.Context) map[string]interface{} {
			v := ctx.Value(ck)
			return map[string]interface{}{ck: v}
		})
		l.Info(ctx, "test_console", map[string]interface{}{"foo": []int{1, 2, 3}})
		So(buf.String(), ShouldContainSubstring, "foo")
	})
}

func TestSingleLog_Normal(t *testing.T) {
	Convey("test hade single log normal case", t, func() {
		basePath := tests.BasePath
		file := "hade_normal.log"

		c := framework.NewHadeContainer()
		c.Singleton(&app.HadeAppProvider{BasePath: basePath})
		c.Singleton(&env.HadeEnvProvider{})
		c.Singleton(&config.FakeConfigProvider{
			FileName: "log",
			Content:  []byte("driver: single\nfile: " + file),
		})
		app := c.MustMake(contract.AppKey).(contract.App)
		folder := app.LogPath()

		err := c.Singleton(&HadeLogServiceProvider{})
		So(err, ShouldBeNil)

		l := c.MustMake(contract.LogKey).(contract.SingleFileLog)
		// check file exist first
		l.Info(context.Background(), "test_single", map[string]interface{}{"foo": "1"})
		f := filepath.Join(folder, file)
		defer os.Remove(f)
		fd, err := os.Stat(f)
		So(err, ShouldBeNil)
		So(fd.Size(), ShouldBeGreaterThan, 0)
	})
}

func TestRotateLog_Normal(t *testing.T) {
	Convey("test hade rotate log normal case", t, func() {
		basePath := tests.BasePath
		file := "hade_normal.log"

		c := framework.NewHadeContainer()
		c.Singleton(&app.HadeAppProvider{BasePath: basePath})
		c.Singleton(&env.HadeEnvProvider{})
		c.Singleton(&config.FakeConfigProvider{
			FileName: "log",
			Content:  []byte("driver: rotate\nfile: " + file + "\nmax_files: 2\ndate_format: \"%Y%m%d\""),
		})
		app := c.MustMake(contract.AppKey).(contract.App)
		folder := app.LogPath()

		err := c.Singleton(&HadeLogServiceProvider{})
		So(err, ShouldBeNil)

		l := c.MustMake(contract.LogKey).(contract.RotatingFileLog)
		// check file exist first
		l.Info(context.Background(), "test_rotate", map[string]interface{}{"foo": "123"})
		f := filepath.Join(folder, file)
		f2 := filepath.Join(folder, file+"."+time.Now().Format("20060102"))
		defer os.Remove(f)
		defer os.Remove(f2)
		_, err = os.Stat(f)
		So(err, ShouldBeNil)
		fd2, err := os.Stat(f2)
		So(err, ShouldBeNil)
		So(fd2.Size(), ShouldBeGreaterThan, 0)
	})
}

func TestTraceLog_Normal(t *testing.T) {
	Convey("test hade single log normal case", t, func() {
		basePath := tests.BasePath

		c := framework.NewHadeContainer()
		c.Singleton(&app.HadeAppProvider{BasePath: basePath})
		c.Singleton(&env.HadeEnvProvider{})
		c.Singleton(&id.HadeIDProvider{})
		c.Singleton(&trace.HadeTraceProvider{})
		tracer := c.MustMake(contract.TraceKey).(contract.Trace)

		err := c.Singleton(&HadeLogServiceProvider{})
		So(err, ShouldBeNil)

		var buf bytes.Buffer
		l := c.MustMake(contract.LogKey).(contract.ConsoleLog)
		l.SetOutput(&buf)
		// check file exist first
		newCtx := tracer.WithTrace(context.Background(), tracer.NewTrace())

		l.Info(newCtx, "test_single", map[string]interface{}{
			"a": "test",
		})
		So(buf.String(), ShouldContainSubstring, contract.TraceKeyTraceID)

	})
}
