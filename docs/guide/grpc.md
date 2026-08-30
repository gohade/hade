# grpc支持

hade框架增加了对grpc的支持，grpc本质上是一个rpc远程调用，使用protobuf作为数据传输格式，使用http2作为传输协议。
hade框架依赖于grpc的go语言实现库: google.golang.org/grpc， 提供了如下命令行工具：

- `hade grpc start` 启动grpc服务
- `hade grpc stop` 停止grpc服务
- `hade grpc restart` 重启grpc服务
- `hade grpc state` 查看grpc服务状态

## 如何用hade创建一个grpc服务

### 1. 创建proto文件

在项目的app/grpc/proto/目录下创建一个proto文件，例如：`proto/helloworld.proto`，内容如下：

```proto
syntax = "proto3";

option go_package = "examples/helloworld";
option java_multiple_files = true;
option java_package = "io.grpc.examples.helloworld";
option java_outer_classname = "HelloWorldProto";

package helloworld;

// The greeting service definition.
service Greeter {
    // Sends a greeting
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// The request message containing the user's name.
message HelloRequest {
    string name = 1;
}

// The response message containing the greetings
message HelloReply {
    string message = 1;
}


```

### 2. 生成go文件

使用grpc官网提供的proto工具生成对应的go文件。

关于proto工具的使用，网上有很多文章了。这里就简要列下mac端的安装命令。

```bash
brew install protobuf

go install google.golang.org/protobuf/cmd/protoc-gen-go

go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

生成go文件的命令如下：

```bash
protoc --go_out=./app/grpc/proto/ --go-grpc_out=./app/grpc/proto/ ./app/grpc/proto/helloworld.proto

```

我们可以看到，在目录`app/grpc/proto/`下生成了文件

```bash
examples/helloworld/helloworld.pb.go
examples/helloworld/helloworld_grpc.pb.go
```

### 3. 创建服务

我们要实现pb中定义的这个服务：helloworld.GreeterServer

在 app/grpc/service/ 目录下创建一个文件：`app/grpc/service/helloworld/service.go`，内容如下：

```go
package helloworld

import (
    "context"
    "log"

    pb "github.com/gohade/hade/app/grpc/proto/examples/helloworld"
)

// Server is used to implement helloworld.GreeterServer.
type Server struct {
    pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    log.Printf("Received: %v", in.GetName())
    return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
```

### 4. 注册服务

在 app/grpc/kernel.go 文件中注册服务

```go
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

```

这里我们不仅注册了服务，还注册了反射服务，这样我们就可以使用grpcurl工具来测试我们的grpc服务了。

### 5. 启动服务

先编译自身项目 `hade build self`

在项目根目录下执行命令：`hade grpc start`，启动grpc服务。

```shell
➜  hade git:(dev/feature-grpc) ✗ ./hade grpc start
成功启动进程: hade grpc
进程pid: 96290
监听地址: grpc://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
```

你也可以通过命令 `hade grpc start -d` 启动服务，这样服务会在后台运行。

默认端口是8888，你可以通过命令行参数 `--address` 来指定端口。

```shell
hade git:(dev/feature-grpc) ✗ ./hade grpc start --address=:8777 -d
成功启动进程: hade grpc
进程pid: 97685
监听地址: grpc://localhost:8777
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
```

### 6. 测试服务

假设我们现在用 daemon 方式启动了一个grpc进程在8777端口

我们要使用grpcurl工具来测试我们的grpc服务。

grpcurl 是 Go 语言开源社区开发的工具，需要手工安装：

```shell
$ go get github.com/fullstorydev/grpcurl
$ go install github.com/fullstorydev/grpcurl/cmd/grpcurl
```

```shell
➜  hade git:(dev/feature-grpc) ✗ grpcurl -plaintext localhost:8777 list
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
helloworld.Greeter
```

可以看到，我们的服务已经注册成功了。

我们来测试一下我们的服务：

```shell
➜  hade git:(dev/feature-grpc) ✗ grpcurl -plaintext -d '{"name": "hade"}' localhost:8777 helloworld.Greeter/SayHello
{
  "message": "Hello hade"
}
```

### 7. 查看grpc服务状态

我们可以通过命令 `hade grpc state` 来查看grpc服务的状态。

```shell
➜  hade git:(dev/feature-grpc) ✗ ./hade grpc state
grpc服务已经启动, pid: 97685
```

### 8. 停止grpc服务

我们可以通过命令 `hade grpc stop` 来停止grpc服务。

```shell
➜  hade git:(dev/feature-grpc) ✗ ./hade grpc stop
停止进程: 97685
```

### 9. 重启grpc服务

我们可以通过命令 `hade grpc restart` 来重启grpc服务。

```shell
➜  hade git:(dev/feature-grpc) ✗ ./hade grpc restart
结束进程成功:1590
成功启动进程: hade grpc
进程pid: 1622
监听地址: grpc://localhost:8888
基础路径: /Users/jianfengye/Documents/workspace/gohade/hade/
日志路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/log
运行路径: /Users/jianfengye/Documents/workspace/gohade/hade/storage/runtime
配置路径: /Users/jianfengye/Documents/workspace/gohade/hade/config
```


