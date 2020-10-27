package ssh

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hade/framework/contract"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type HadeSSH struct {
	config *contract.SSHConfig

	client     *ssh.Client
	session    *ssh.Session
	sftpClient *sftp.Client
}

func NewHadeSSH(params ...interface{}) (interface{}, error) {
	config := params[0].(*contract.SSHConfig)
	if config == nil {
		return nil, errors.New("config error")
	}
	// 连接
	auth := []ssh.AuthMethod{}
	if config.Password != "" {
		auth = append(auth, ssh.Password(config.Password))
	} else if config.RsaKey != "" {
		key, err := ioutil.ReadFile(config.RsaKey)
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	conf := &ssh.ClientConfig{
		User:    config.User,
		Auth:    auth,
		Timeout: time.Duration(config.Timeout) * time.Millisecond,
	}
	client, err := ssh.Dial("tcp", config.Host+":"+config.Port, conf)
	if err != nil {
		return nil, err
	}
	ss, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	sftpClient, err := sftp.NewClient(client, nil)
	if err != nil {
		return nil, err
	}

	return &HadeSSH{config: config, client: client, session: ss, sftpClient: sftpClient}, nil
}

// 运行shell，并输出结果
func (sh *HadeSSH) Run(shell string) ([]byte, error) {
	var b bytes.Buffer     // import "bytes"
	sh.session.Stdout = &b // get output
	if err := sh.session.Run(shell); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// 将本地文件放到远端, 必须确保目录都存在
func (sh *HadeSSH) Upload(src, dist string) error {
	// 用来测试的本地文件路径 和 远程机器上的文件夹
	srcFile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := sh.sftpClient.Create(dist)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	buf := make([]byte, 1024)
	for {
		n, err := srcFile.Read(buf)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := dstFile.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

// 将远端文件放到本地
func (sh *HadeSSH) Download(src, dist string) error {
	// 用来测试的本地文件路径 和 远程机器上的文件夹
	distFile, err := os.Open(dist)
	if err != nil {
		log.Fatal(err)
	}
	defer distFile.Close()

	srcFile, err := sh.sftpClient.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	if _, err := srcFile.WriteTo(distFile); err != nil {
		return err
	}
	return nil
}

// dir 前后都包含分割线
func (sh *HadeSSH) UploadDir(src, dist string, clean bool) error {
	if !strings.HasSuffix(src, "/") {
		src = src + "/"
	}
	if !strings.HasSuffix(dist, "/") {
		dist = dist + "/"
	}

	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// check dir exist in remote
			if strings.HasSuffix(path, "/") {
				path = path + "/"
			}
			distPath := dist + strings.Replace(path, src, "", 1)
			_, err := sh.sftpClient.Stat(distPath)
			if err == os.ErrNotExist {
				sh.sftpClient.Mkdir(distPath)
			} else {
				return err
			}
		} else {
			distPath := dist + strings.Replace(path, src, "", 1)
			if err := sh.Upload(path, distPath); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

func (sh *HadeSSH) DownloadDir(src, dist string) error {
	if !strings.HasSuffix(src, "/") {
		src = src + "/"
	}
	if !strings.HasSuffix(dist, "/") {
		dist = dist + "/"
	}
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// check dir exist in remote
			if strings.HasSuffix(path, "/") {
				path = path + "/"
			}
			distPath := dist + strings.Replace(path, src, "", 1)
			stat, err := os.Stat(distPath)
			if err == os.ErrNotExist {
				if err := os.Mkdir(distPath, stat.Mode()); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			distPath := dist + strings.Replace(path, src, "", 1)
			if err := sh.Download(path, distPath); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}
