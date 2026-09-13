# hade:ssh

## 说明

hade:ssh 是提供SSH服务的服务，可以用于获取ssh连接实例。

## 配置

使用 hade:ssh 服务之前必须确保正确配置 ssh，配置文件在 `config/[env]/ssh.yaml`。

以下是一个配置的例子：

```yaml
timeout: 3s
network: tcp
web-01:
    host: 111.222.333.444 # ip地址
    port: 22 # 端口
    username: yejianfeng # 用户名
    password: "123456" # 密码
web-02:
    network: tcp
    host: localhost # ip地址
    port: 3306 # 端口
    username: jianfengye # 用户名
    rsa_key: "/Users/user/.ssh/id_rsa"
    known_hosts: "/Users/user/.ssh/known_hosts"


#host: localhost # ip地址
#port: 3306 # 端口
#username: jianfengye # 用户名
#password: "123456789" # 密码
#rsa_key: "~/.ssh/id_rsa"
#timeout: 1000
#network: tcp

```

## 使用方法

```
// SSHService 表示一个ssh服务
type SSHService interface {
	// GetClient 获取ssh连接实例
	GetClient(option ...SSHOption) (*ssh.Client, error)
}

```
