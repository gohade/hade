package contract

import (
	"fmt"
	"github.com/gohade/hade/framework"
	"golang.org/x/crypto/ssh"
)

const SSHKey = "hade:ssh"

// SSHOption 代表初始化的时候的选项
type SSHOption func(container framework.Container, config *SSHConfig) error

// SSHService 表示一个ssh服务
type SSHService interface {
	// GetClient 获取ssh连接实例
	GetClient(option ...SSHOption) (*ssh.Client, error)
}

// SSHConfig 为hade定义的SSH配置结构
type SSHConfig struct {
	NetWork string
	Host    string
	Port    string
	*ssh.ClientConfig
}

// UniqKey 用来唯一标识一个SSHConfig配置
func (config *SSHConfig) UniqKey() string {
	return fmt.Sprintf("%v_%v_%v", config.Host, config.Port, config.User)
}
