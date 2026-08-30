# hade:redis

## 说明

hade:redis 是提供redis服务的服务，可以用于获取redis连接实例。

## 配置

使用 hade:redis 之前需要确保已经正确配置了redis服务。配置文件为 config/[env]/redis.yaml。
以下是一个配置的例子：

```yaml
timeout: 10s # 连接超时
read_timeout: 2s # 读超时
write_timeout: 2s # 写超时

write:
    host: 127.0.0.1 # ip地址
    port: 6379 # 端口
    db: 0 #db
    timeout: 10s # 连接超时
    read_timeout: 2s # 读超时
    write_timeout: 2s # 写超时

```

## 使用方法

```
// RedisService 表示一个redis服务
type RedisService interface {
	// GetClient 获取redis连接实例
	GetClient(option ...RedisOption) (*redis.Client, error)
}
```
