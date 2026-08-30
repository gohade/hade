package contract

import (
	"net/http"

	"google.golang.org/grpc"
)

const KernelKey = "hade:kernel"

// Kernel 接口提供框架最核心的结构
type Kernel interface {
	// HttpEngine 提供gin的Engine结构
	HttpEngine() http.Handler
	// GrpcEngine 提供grpc的Engine结构
	GrpcEngine() *grpc.Server
	// AgentEngine 提供 Agent API 的 Engine 结构
	AgentEngine() http.Handler
}
