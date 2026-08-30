//go:build (!darwin && !linux) || pidlockstub

package agent

import (
	"errors"
	"os"
)

// errPIDLockUnsupported 是所有平台原语在非 Unix 平台上的统一返回。
// Agent 的进程所有权依赖 flock 与 fd 复制语义，缺一不可，因此这里明确拒绝，
// 而不是给出一个看起来能用的降级实现。
var errPIDLockUnsupported = errors.New("agent 进程锁与 fd 复制仅支持 Darwin/Linux")

func tryLockFile(file *os.File) (bool, error) {
	return false, errPIDLockUnsupported
}

func unlockFile(file *os.File) error {
	return errPIDLockUnsupported
}

func duplicateFD(fd int) (int, error) {
	return -1, errPIDLockUnsupported
}

func closeFD(fd int) error {
	return errPIDLockUnsupported
}
