package contract

import (
    "context"
    "github.com/gohade/hade/framework"
    "time"
)

const CacheKey = "hade:cache"

// RememberFunc 缓存的Remember方法使用，Cache-Aside模式对应的对象生成方法
type RememberFunc func(ctx context.Context, container framework.Container) (interface{}, error)

// CacheService 缓存服务
type CacheService interface {
    // Get 获取某个key对应的值
    Get(ctx context.Context, key string) (string, error)
    // GetObj 获取某个key对应的对象, 对象必须实现 https://pkg.go.dev/encoding#BinaryUnMarshaler
    GetObj(ctx context.Context, key string, model interface{}) error
    // GetMany 获取某些key对应的值
    GetMany(ctx context.Context, keys []string) (map[string]string, error)

    // Set 设置某个key和值到缓存，带超时时间
    Set(ctx context.Context, key string, val string, timeout time.Duration) error
    // SetObj 设置某个key和对象到缓存, 对象必须实现 https://pkg.go.dev/encoding#BinaryMarshaler
    SetObj(ctx context.Context, key string, val interface{}, timeout time.Duration) error
    // SetMany 设置多个key和值到缓存
    SetMany(ctx context.Context, data map[string]string, timeout time.Duration) error
    // SetForever 设置某个key和值到缓存，不带超时时间
    SetForever(ctx context.Context, key string, val string) error
    // SetForeverObj 设置某个key和对象到缓存，不带超时时间，对象必须实现 https://pkg.go.dev/encoding#BinaryMarshaler
    SetForeverObj(ctx context.Context, key string, val interface{}) error

    // SetTTL 设置某个key的超时时间
    SetTTL(ctx context.Context, key string, timeout time.Duration) error
    // GetTTL 获取某个key的超时时间
    GetTTL(ctx context.Context, key string) (time.Duration, error)

    // Remember 实现缓存的Cache-Aside模式, 先去缓存中根据key获取对象，如果有的话，返回，如果没有，调用RememberFunc 生成
    Remember(ctx context.Context, key string, timeout time.Duration, rememberFunc RememberFunc, model interface{}) error

    // Calc 往key对应的值中增加step计数
    Calc(ctx context.Context, key string, step int64) (int64, error)
    // Increment 往key对应的值中增加1
    Increment(ctx context.Context, key string) (int64, error)
    // Decrement 往key对应的值中减去1
    Decrement(ctx context.Context, key string) (int64, error)

    // Del 删除某个key
    Del(ctx context.Context, key string) error
    // DelMany 删除某些key
    DelMany(ctx context.Context, keys []string) error
}
