package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gohade/hade/framework/util"
)

const processPollInterval = 100 * time.Millisecond

type processOperations struct {
	command   func(int) (string, error)
	terminate func(int) error
	exists    func(int) bool
	sleep     func(time.Duration)
}

func defaultProcessOperations() processOperations {
	return processOperations{
		command: func(pid int) (string, error) {
			output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(output)), nil
		},
		terminate: util.KillProcess,
		exists:    util.CheckProcessExist,
		sleep:     time.Sleep,
	}
}

func isAgentProcess(command, executable string) bool {
	fields := strings.Fields(command)
	return len(fields) >= 2 &&
		filepath.Base(fields[0]) == filepath.Base(executable) &&
		fields[1] == "agent"
}

func readPIDFile(pidFile string) (int, bool, error) {
	content, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false, err
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return 0, false, nil
	}
	pid, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("无效的 agent pid %q: %w", value, err)
	}
	return pid, true, nil
}

func cleanupPIDFile(pidFile string, currentPID int) error {
	pid, exists, err := readPIDFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !exists || pid != currentPID {
		return nil
	}
	return os.WriteFile(pidFile, []byte{}, 0644)
}

func stopAgentProcess(pidFile, executable string, ops processOperations, wait time.Duration) error {
	pid, exists, err := readPIDFile(pidFile)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	command, err := ops.command(pid)
	if err != nil {
		return fmt.Errorf("无法确认 pid %d 的进程身份，拒绝停止: %w", pid, err)
	}
	if !isAgentProcess(command, executable) {
		return fmt.Errorf("pid %d 进程身份不匹配 agent 服务，拒绝停止", pid)
	}
	if err := ops.terminate(pid); err != nil {
		return err
	}

	attempts := int(wait / processPollInterval)
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		if !ops.exists(pid) {
			return cleanupPIDFile(pidFile, pid)
		}
		ops.sleep(processPollInterval)
	}
	if ops.exists(pid) {
		return fmt.Errorf("等待 agent 服务进程 %d 退出超时", pid)
	}
	return cleanupPIDFile(pidFile, pid)
}
