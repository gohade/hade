// Copyright 2021 jianfengye.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
package middleware

import (
	"log"
	"time"

	"github.com/gohade/hade/framework/gin"
)

// Cost struct
type Cost struct {
}

// Func ...
func (c *Cost) Func() gin.HandlerFunc {
	// 使用函数回调
	return func(c *gin.Context) {
		// 记录开始时间
		start := time.Now()

		log.Printf("api uri start: %v", c.Request.RequestURI)
		// 使用next执行具体的业务逻辑
		c.Next()

		// 记录结束时间
		end := time.Now()
		cost := end.Sub(start)
		log.Printf("api uri end: %v, cost: %v", c.Request.RequestURI, cost.Seconds())

	}
}
