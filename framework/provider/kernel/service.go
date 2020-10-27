package kernel

import (
	"hade/framework/gin"
)

type HadeKernelService struct {
	engine *gin.Engine
}

func NewHadeKernelService(params ...interface{}) (interface{}, error) {
	httpEngine := params[0].(*gin.Engine)
	return &HadeKernelService{engine: httpEngine}, nil
}

func (s *HadeKernelService) HttpEngine() *gin.Engine {
	return s.engine
}
