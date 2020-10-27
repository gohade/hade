package contract

const SSHKey = "hade:ssh"

// SSH 相关接口
type SSH interface {
	// 运行shell，并输出结果
	Run(shell string) ([]byte, error)
	// 将本地文件放到远端
	Upload(src, dist string) error
	// 将远端文件放到本地
	Download(src, dist string) error
	// 将本地文件夹放到远端
	UploadDir(src, dist string) error

	DownloadDir(src, dist string) error
}

// SSH的连接配置
// 如果设置了Password，优先走Password模式，否则走RSAKey
// 密码模式和rsa key模式必须至少有一个
type SSHConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	RsaKey   string
	Timeout  int
}
