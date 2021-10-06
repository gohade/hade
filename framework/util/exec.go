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

// CheckProcessExist Will return true if the process with PID exists.
func CheckProcessExist(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}
	return true
}
