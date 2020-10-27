package http

import (
	"github.com/gohade/hade/app/http/module/demo"
	"github.com/gohade/hade/framework/gin"
)

func Routes(r *gin.Engine) {

	r.Static("/dist/", "./dist/")

	demo.Register(r)
}
