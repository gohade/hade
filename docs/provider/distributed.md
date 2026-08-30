# hade:distributed

## 说明

hade:distributed 是提供分布式选举的服务，可以用于分布式锁，分布式任务调度等场景。当分布式集群中有需要选举出一个节点来执行任务时，可以使用此服务。

目前仅支持本地多进程的文件实现，后续会支持redis等分布式存储。

## 使用方法

```go

// DistributedKey 定义字符串凭证
const DistributedKey = "hade:distributed"

// Distributed 分布式服务
type Distributed interface {
// Select 分布式选择器, 所有节点对某个服务进行抢占，只选择其中一个节点
// ServiceName 服务名字
// appID 当前的AppID
// holdTime 分布式选择器hold住的时间
// 返回值
// selectAppID 分布式选择器最终选择的App
// err 异常才返回，如果没有被选择，不返回err
Select(serviceName string, appID string, holdTime time.Duration) (selectAppID string, err error)
}

```
