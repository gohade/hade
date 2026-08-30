# hade:kernel

## 说明

hade:kernel 是提供框架最核心的结构，包括http和grpc的Engine结构。

## 使用方法

```

const KernelKey = "hade:kernel"

// Kernel 接口提供框架最核心的结构
type Kernel interface {
    // HttpEngine 提供gin的Engine结构
    HttpEngine() http.Handler
    // GrpcEngine 提供grpc的Engine结构
    GrpcEngine() *grpc.Server
}

```
