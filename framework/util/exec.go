//go:build !windows
// +build !windows

package util

import (
	"os"
	"syscall"
)

// GetExecDirectory 获取当前执行程序目录
func GetExecDirectory() string {
	file, err := os.Getwd()
	if err == nil {
		return file + "/"
	}
	return ""
}

// GetRootDirectory 获取当前项目根目录
func GetRootDirectory() (string, error) {
	executable, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(executable)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".go-root")); err == nil {
			return dir, nil
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	return "", fmt.Errorf("unable to find project root")
}

// CheckProcessExist 检查进程pid是否存在，如果存在的话，返回true
func CheckProcessExist(pid int) bool {
	// 查询这个pid
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 给进程发送signal 0, 如果返回nil，代表进程存在, 否则进程不存在
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}
	return true
}

// KillProcess kill process by pid
func KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
