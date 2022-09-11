package distributed

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// LocalDistributedService 代表hade框架的App实现
type LocalDistributedService struct {
	container framework.Container // 服务容器
}

// NewLocalDistributedService 初始化本地分布式服务
func NewLocalDistributedService(params ...interface{}) (interface{}, error) {
	if len(params) != 1 {
		return nil, errors.New("param error")
	}

	// 有两个参数，一个是容器，一个是baseFolder
	container := params[0].(framework.Container)
	return &LocalDistributedService{container: container}, nil
}

// Select 为分布式选择器
func (s LocalDistributedService) Select(serviceName string, appID string, holdTime time.Duration) (selectAppID string, err error) {
	appService := s.container.MustMake(contract.AppKey).(contract.App)
	runtimeFolder := appService.RuntimeFolder()
	lockFile := filepath.Join(runtimeFolder, "disribute_"+serviceName)

	// 打开文件锁
	lock, err := os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	h, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return "", err
	}
	defer syscall.FreeLibrary(h)

	addr, err := syscall.GetProcAddress(h, "LockFile")
	if err != nil {
		return "", err
	}
	unlockAddr, err := syscall.GetProcAddress(h, "UnlockFile")
	if err != nil {
		return "", err
	}
	// 尝试抢占文件锁
	r0, _, _ := syscall.Syscall6(addr, 5, lock.Fd(), 0, 0, 0, 1, 0)
	if 0 == int(r0) {
		// 抢不到
		// 读取被选择的appid
		selectAppIDByt, err := ioutil.ReadAll(lock)
		if err != nil {
			return "", err
		}
		return string(selectAppIDByt), err
	}
	// 抢到了

	// 在一段时间内，选举有效，其他节点在这段时间不能再进行抢占
	go func() {
		defer func() {
			syscall.Syscall6(unlockAddr, 5, lock.Fd(), 0, 0, 0, 1, 0)
			// 释放文件
			lock.Close()
			// 删除文件锁对应的文件
			os.Remove(lockFile)
		}()
		// 创建选举结果有效的计时器
		timer := time.NewTimer(holdTime)
		// 等待计时器结束
		<-timer.C
	}()

	// 这里已经是抢占到了，将抢占到的appID写入文件
	if _, err := lock.WriteString(appID); err != nil {
		return "", err
	}
	return appID, nil
}
