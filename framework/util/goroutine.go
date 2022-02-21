package util

import (
	"bytes"
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"runtime/debug"
	"sync"
)

// SafeGo 进行安全的goroutine调用
// 第一个参数是container容器，要求容器中必须绑定日志服务
// 第二个参数是匿名函数handler, 进行最终的业务逻辑
// SafeGo 函数并不会返回error，handler的error和panic都会进入hade的日志服务
func SafeGo(container framework.Container, handler func() error) {
	logger := container.MustMake(contract.LogKey).(contract.Log)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				buf := debug.Stack()
				buf = bytes.ReplaceAll(buf, []byte("\n"), []byte("\\n"))
				logger.Error(context.Background(), "safe go handler panic", map[string]interface{}{
					"stack": string(buf),
					"err":   e,
				})
			}
		}()
		if e := handler(); e != nil {
			logger.Error(context.Background(), "safe go handler error", map[string]interface{}{
				"err": e,
			})
		}
	}()
}

// SafeGoAndWait 进行并发安全并行调用
// 第一个参数是container容器，要求容器中必须绑定日志服务
// 第二个参数是匿名函数handlers数组, 进行最终的业务逻辑
// 返回handlers中任何一个错误（如果handlers中有业务逻辑返回错误）
func SafeGoAndWait(container framework.Container, handlers ...func() error) error {
	var (
		wg   sync.WaitGroup
		once sync.Once
		err  error
	)

	logger := container.MustMake(contract.LogKey).(contract.Log)
	for _, f := range handlers {
		wg.Add(1)
		go func(handler func() error) {
			defer func() {
				if err := recover(); err != nil {
					buf := debug.Stack()
					buf = bytes.ReplaceAll(buf, []byte("\n"), []byte("\\n"))
					logger.Error(context.Background(), "panic", map[string]interface{}{
						"stack": string(buf),
						"err":   err,
					})
				}
				wg.Done()
			}()
			if e := handler(); e != nil {
				logger.Error(context.Background(), "SafeGo Error: %v", map[string]interface{}{
					"err": e,
				})

				once.Do(func() {
					err = e
				})
			}
		}(f)
	}
	wg.Wait()
	return err
}
