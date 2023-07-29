package kernel

import (
	"net/http"

	"github.com/gohade/hade/framework/gin"
	"google.golang.org/grpc"
)

// HadeKernelService 引擎服务
type HadeKernelService struct {
	httpEngine *gin.Engine
	grpcEngine *grpc.Server
}

// NewHadeKernelService 初始化引擎服务实例
func NewHadeKernelService(params ...interface{}) (interface{}, error) {
	httpEngine := params[0].(*gin.Engine)
	grpcEngine := params[1].(*grpc.Server)
	return &HadeKernelService{httpEngine: httpEngine, grpcEngine: grpcEngine}, nil
}

// HttpEngine 返回web引擎
func (s *HadeKernelService) HttpEngine() http.Handler {
	return s.httpEngine
}

// GrpcEngine 返回grpc引擎
func (s *HadeKernelService) GrpcEngine() *grpc.Server {
	return s.grpcEngine
}
