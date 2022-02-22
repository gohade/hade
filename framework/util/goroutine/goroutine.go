package goroutine

import (
	"bytes"
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"log"
	"runtime/debug"
	"sync"
)

// SafeGo 进行安全的goroutine调用
// 第一个参数是context接口，如果还实现了Container接口，且绑定了日志服务，则使用日志服务
// 第二个参数是匿名函数handler, 进行最终的业务逻辑
// SafeGo 函数并不会返回error，panic都会进入hade的日志服务
func SafeGo(ctx context.Context, handler func()) {
	var logger contract.Log
	if container, ok := ctx.(framework.Container); ok {
		if container.IsBind(contract.LogKey) {
			logger = container.MustMake(contract.LogKey).(contract.Log)
		}
	}

	go func() {
		defer func() {
			if e := recover(); e != nil {
				buf := debug.Stack()
				buf = bytes.ReplaceAll(buf, []byte("\n"), []byte("\\n"))
				if logger != nil {
					logger.Error(ctx, "safe go handler panic", map[string]interface{}{
						"stack": string(buf),
						"err":   e,
					})
				} else {
					log.Printf("panic\t%v\t%s", e, buf)
				}
			}
		}()
		handler()
	}()
}

// SafeGoAndWait 进行并发安全并行调用
// 第一个参数是context接口，如果还实现了Container接口，且绑定了日志服务，则使用日志服务
// 第二个参数是匿名函数handlers数组, 进行最终的业务逻辑
// 返回handlers中任何一个错误（如果handlers中有业务逻辑返回错误）
func SafeGoAndWait(ctx context.Context, handlers ...func() error) error {
	var (
		wg     sync.WaitGroup
		once   sync.Once
		err    error
		logger contract.Log
	)
	if container, ok := ctx.(framework.Container); ok {
		if container.IsBind(contract.LogKey) {
			logger = container.MustMake(contract.LogKey).(contract.Log)
		}
	}

	for _, f := range handlers {
		wg.Add(1)
		go func(handler func() error) {
			defer func() {
				if err := recover(); err != nil {
					buf := debug.Stack()
					buf = bytes.ReplaceAll(buf, []byte("\n"), []byte("\\n"))
					if logger != nil {
						logger.Error(ctx, "panic", map[string]interface{}{
							"stack": string(buf),
							"err":   err,
						})
					} else {
						log.Printf("panic\t%v\t%s", err, buf)
					}
				}
				wg.Done()
			}()
			if e := handler(); e != nil {
				once.Do(func() {
					err = e
				})
			}
		}(f)
	}
	wg.Wait()
	return err
}
