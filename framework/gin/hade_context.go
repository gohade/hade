// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package gin

import (
	"context"
)

// BaseContext 获取基础的context
func (ctx *Context) BaseContext() context.Context {
	return ctx.Request.Context()
}

// =======
// context 实现container的几个封装, 所以hade.gin.context 也实现了 hade.Container接口
// =======

// Make 实现make的封装
func (ctx *Context) Make(key string) (interface{}, error) {
	return ctx.container.Make(key)
}

// MustMake 实现mustMake的封装
func (ctx *Context) MustMake(key string) interface{} {
	return ctx.container.MustMake(key)
}

// MakeNew 实现makenew的封装
func (ctx *Context) MakeNew(key string, params []interface{}) (interface{}, error) {
	return ctx.container.MakeNew(key, params)
}

// IsBind 空实现
func (ctx *Context) IsBind(key string) bool {
	return ctx.container.IsBind(key)
}
