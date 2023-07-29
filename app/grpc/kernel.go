package grpc

import (
	helloworldgen "github.com/gohade/hade/app/grpc/proto/examples/helloworld"
	"github.com/gohade/hade/app/grpc/service/helloworld"
	"github.com/gohade/hade/framework"
	pkggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewGrpcEngine 创建了一个绑定了路由的Web引擎
func NewGrpcEngine(container framework.Container) (*pkggrpc.Server, error) {

	s := pkggrpc.NewServer()

	// 这里进行服务注册
	helloworldgen.RegisterGreeterServer(s, &helloworld.Server{})
	reflection.Register(s)

	return s, nil
}
