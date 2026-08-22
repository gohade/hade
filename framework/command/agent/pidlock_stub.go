//go:build !darwin && !linux

package agent

import (
	"errors"
	"os"
)

func tryLockFile(file *os.File) (bool, error) {
	return false, errors.New("agent pid 文件锁仅支持 Darwin/Linux")
}

func unlockFile(file *os.File) error {
	return errors.New("agent pid 文件锁仅支持 Darwin/Linux")
}
