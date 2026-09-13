package gen

import (
	"github.com/gohade/hade/framework/gin"
)

type StudentApi struct {
}

func NewApi() *StudentApi {
	return &StudentApi{}
}

func Register(r *gin.Engine) error {
	api := NewApi()

	r.GET("/student/show", api.Show)
	r.GET("/student/list", api.List)
	r.POST("/student/create", api.Create)
	r.POST("/student/update", api.Update)
	r.POST("/student/delete", api.Delete)
	return nil
}
