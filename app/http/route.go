package http

import (
	"hade/app/http/module/demo"
	"hade/framework/gin"
)

func Routes(r *gin.Engine) {

	r.Static("/dist/", "./dist/")

	demo.Register(r)
}
