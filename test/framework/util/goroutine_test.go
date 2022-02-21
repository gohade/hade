package util

import (
	"errors"
	"github.com/gohade/hade/framework/provider/log"
	"github.com/gohade/hade/framework/util"
	tests "github.com/gohade/hade/test"
	"testing"
	"time"
)

func TestSafeGo(t *testing.T) {
	container := tests.InitBaseContainer()
	container.Bind(&log.HadeTestingLogProvider{})

	errStr := "safe go test error"
	util.SafeGo(container, func() error {
		time.Sleep(1 * time.Second)
		return errors.New(errStr)
	})
	t.Log("safe go main start")
	time.Sleep(2 * time.Second)
	t.Log("safe go main end")

	util.SafeGo(container, func() error {
		time.Sleep(1 * time.Second)
		panic("safe go test panic")
	})
	t.Log("safe go2 main start")
	time.Sleep(2 * time.Second)
	t.Log("safe go2 main end")

}

func TestSafeGoAndWait(t *testing.T) {
	container := tests.InitBaseContainer()
	container.Bind(&log.HadeTestingLogProvider{})

	errStr := "safe go test error"
	t.Log("safe go and wait start", time.Now().String())

	err := util.SafeGoAndWait(container, func() error {
		time.Sleep(1 * time.Second)
		return errors.New(errStr)
	}, func() error {
		time.Sleep(2 * time.Second)
		return nil
	}, func() error {
		time.Sleep(3 * time.Second)
		return nil
	})
	t.Log("safe go and wait end", time.Now().String())

	if err == nil {
		t.Error("err not be nil")
	} else if err.Error() != errStr {
		t.Error("err content not same")
	}

	// panic error
	err = util.SafeGoAndWait(container, func() error {
		time.Sleep(1 * time.Second)
		return errors.New(errStr)
	}, func() error {
		time.Sleep(2 * time.Second)
		panic("test2")
	}, func() error {
		time.Sleep(3 * time.Second)
		return nil
	})
	if err == nil {
		t.Error("err not be nil")
	} else if err.Error() != errStr {
		t.Error("err content not same")
	}
}
