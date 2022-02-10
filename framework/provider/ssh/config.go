package ssh

import (
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"io/ioutil"
	"time"
)

// GetBaseConfig 读取database.yaml根目录结构
func GetBaseConfig(c framework.Container) *contract.SSHConfig {
	logService := c.MustMake(contract.LogKey).(contract.Log)
	config := &contract.SSHConfig{ClientConfig: &ssh.ClientConfig{Auth: []ssh.AuthMethod{}, HostKeyCallback: ssh.InsecureIgnoreHostKey()}}
	opt := WithConfigPath("ssh")
	err := opt(c, config)
	if err != nil {
		// 直接使用logService来打印错误信息
		logService.Error(context.Background(), "parse cache config error", map[string]interface{}{
			"err": err,
		})
		return nil
	}
	return config
}

// WithConfigPath 加载配置文件地址
func WithConfigPath(configPath string) contract.SSHOption {
	return func(container framework.Container, config *contract.SSHConfig) error {
		configService := container.MustMake(contract.ConfigKey).(contract.Config)
		logService := container.MustMake(contract.LogKey).(contract.Log)
		conf := configService.GetStringMapString(configPath)
		// 读取config配置
		/*
		   		    #host: localhost # ip地址
		               #port: 3306 # 端口
		               #username: jianfengye # 用户名
		               #password: "123456789" # 密码
		               #timeout: 1000
		               #network: tcp
		                #rsa_key: "/Users/user/.ssh/id_rsa"
		                #known_hosts: "/Users/user/.ssh/known_hosts"
		*/
		if network, ok := conf["network"]; ok {
			config.NetWork = network
		}

		if host, ok := conf["host"]; ok {
			config.Host = host
		}
		if port, ok := conf["port"]; ok {
			config.Port = port
		}

		if username, ok := conf["username"]; ok {
			config.User = username
		}

		if password, ok := conf["password"]; ok {
			authPwd := ssh.Password(password)
			config.Auth = append(config.Auth, authPwd)
		}

		if rsaKey, ok := conf["rsa_key"]; ok {
			key, err := ioutil.ReadFile(rsaKey)
			if err != nil {
				logService.Error(context.Background(), "read rsa_key error", map[string]interface{}{
					"key":  rsaKey,
					"path": configPath,
					"err":  err,
				})
			}
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				logService.Error(context.Background(), "create rsa_key signer error", map[string]interface{}{
					"key":  rsaKey,
					"path": configPath,
					"err":  err,
				})
			}
			rsaKeyAuth := ssh.PublicKeys(signer)
			config.Auth = append(config.Auth, rsaKeyAuth)
		}

		if knownHosts, ok := conf["known_host"]; ok {
			hostKeyCallback, err := knownhosts.New(knownHosts)
			if err != nil {
				logService.Error(context.Background(), "knownhosts error", map[string]interface{}{
					"key":  knownHosts,
					"path": configPath,
					"err":  err,
				})
			}
			config.HostKeyCallback = hostKeyCallback
		}

		if timeout, ok := conf["timeout"]; ok {
			t, err := time.ParseDuration(timeout)
			if err != nil {
				return err
			}
			config.Timeout = t
		}

		return nil
	}
}

// WithSSHConfig 表示自行配置redis的配置信息
func WithSSHConfig(f func(options *contract.SSHConfig)) contract.SSHOption {
	return func(container framework.Container, config *contract.SSHConfig) error {
		f(config)
		return nil
	}
}
