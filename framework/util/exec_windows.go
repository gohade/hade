package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetExecDirectory 获取当前执行程序目录
func GetExecDirectory() string {
	file, err := os.Getwd()
	if err == nil {
		return file + "/"
	}
	return ""
}

// CheckProcessExist 检查进程pid是否存在，如果存在的话，返回true
func CheckProcessExist(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 检查输出中是否包含进程名称
	return strings.Contains(string(output), fmt.Sprintf("%d", pid))
}

// KillProcess kill process by pid
func KillProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
