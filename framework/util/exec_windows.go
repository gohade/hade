package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
