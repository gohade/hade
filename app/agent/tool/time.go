package tool

import (
	"context"
	"time"
)

// Time 返回当前 UTC 时间，不访问外部系统。
func Time(_ context.Context, _ string) (string, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}
