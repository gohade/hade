//go:build (darwin || linux) && !pidlockstub

package agent

import (
	"os"
	"syscall"
)

func tryLockFile(file *os.File) (bool, error) {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return false, nil
	}
	return false, err
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

// duplicateFD 复制一个继承来的文件描述符，复制出的 fd 与原 fd 共享同一把
// flock 锁描述，这是校验 daemon authorization 的前提。
func duplicateFD(fd int) (int, error) {
	return syscall.Dup(fd)
}

func closeFD(fd int) error {
	return syscall.Close(fd)
}
