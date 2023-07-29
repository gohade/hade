package grpc

import (
    helloworld2 "github.com/gohade/hade/app/grpc/proto/helloworld"
    "github.com/gohade/hade/app/grpc/service/helloworld"
    "github.com/gohade/hade/framework"
    pkggrpc "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

// NewGrpcEngine 创建了一个绑定了路由的Web引擎
func NewGrpcEngine(container framework.Container) (*pkggrpc.Server, error) {

    // 这里注入各种路由
    s := pkggrpc.NewServer()
    helloworld2.RegisterGreeterServer(s, &helloworld.Server{})
    reflection.Register(s)

    return s, nil
}
