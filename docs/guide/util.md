# 辅助函数

hade提供了一些辅助函数来帮助你更好的进行开发。

## goroutin相关

### SafeGo

SafeGo 这个函数，提供了一种goroutine安全的函数调用方式。主要适用于业务中需要进行开启异步goroutine业务逻辑调用的场景。

```
// SafeGo 进行安全的goroutine调用
// 第一个参数是context接口，如果还实现了Container接口，且绑定了日志服务，则使用日志服务
// 第二个参数是匿名函数handler, 进行最终的业务逻辑
// SafeGo 函数并不会返回error，panic都会进入hade的日志服务
func SafeGo(ctx context.Context, handler func())
```

调用方式参照如下的单元测试用例：

```

func TestSafeGo(t *testing.T) {
    container := tests.InitBaseContainer()
    container.Bind(&log.HadeTestingLogProvider{})

    ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
    goroutine.SafeGo(ctx, func() {
        time.Sleep(1 * time.Second)
        return
    })
    t.Log("safe go main start")
    time.Sleep(2 * time.Second)
    t.Log("safe go main end")

    goroutine.SafeGo(ctx, func() {
        time.Sleep(1 * time.Second)
        panic("safe go test panic")
    })
    t.Log("safe go2 main start")
    time.Sleep(2 * time.Second)
    t.Log("safe go2 main end")

}
```

### SafeGoAndWait

SafeGoAndWait 这个函数，提供安全的多并发调用方式。该函数等待所有函数都结束后才返回。

```
// SafeGoAndWait 进行并发安全并行调用
// 第一个参数是context接口，如果还实现了Container接口，且绑定了日志服务，则使用日志服务
// 第二个参数是匿名函数handlers数组, 进行最终的业务逻辑
// 返回handlers中任何一个错误（如果handlers中有业务逻辑返回错误）
func SafeGoAndWait(ctx context.Context, handlers ...func() error) error
```

调用方式参照如下的单元测试用例：

```

func TestSafeGoAndWait(t *testing.T) {
    container := tests.InitBaseContainer()
    container.Bind(&log.HadeTestingLogProvider{})

    errStr := "safe go test error"
    t.Log("safe go and wait start", time.Now().String())
    ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

    err := goroutine.SafeGoAndWait(ctx, func() error {
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
    err = goroutine.SafeGoAndWait(ctx, func() error {
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

```
