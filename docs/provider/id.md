# hade:id

## 说明

hade:id 是提供分布式ID生成服务，可以为当前服务生成唯一 id。

## 使用方法

```
const IDKey = "hade:id"

type IDService interface {
    NewID() string
}
```
