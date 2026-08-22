package agent

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gohade/hade/framework/util"
)

const processPollInterval = 100 * time.Millisecond

type processOperations struct {
	terminate func(int) error
	sleep     func(time.Duration)
}

func defaultProcessOperations() processOperations {
	return processOperations{
		terminate: util.KillProcess,
		sleep:     time.Sleep,
	}
}

type pidFileOwner struct {
	file   *os.File
	pid    int
	closed bool
}

func acquirePIDFile(path string, pid int) (*pidFileOwner, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	locked, err := tryLockFile(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	if !locked {
		file.Close()
		return nil, fmt.Errorf("agent pid 文件已被其他进程锁定")
	}
	if err := writePID(file, pid); err != nil {
		unlockFile(file)
		file.Close()
		return nil, err
	}
	return &pidFileOwner{file: file, pid: pid}, nil
}

func writePID(file *os.File, pid int) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := file.WriteString(strconv.Itoa(pid)); err != nil {
		return err
	}
	return file.Sync()
}

func readPID(file *os.File) (int, bool, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, false, err
	}
	content, err := io.ReadAll(file)
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

func clearPID(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	_, err := file.Seek(0, io.SeekStart)
	return err
}

func (owner *pidFileOwner) cleanup() error {
	if owner == nil || owner.closed {
		return nil
	}
	pid, exists, readErr := readPID(owner.file)
	if readErr == nil && exists && pid == owner.pid {
		readErr = clearPID(owner.file)
	}
	releaseErr := owner.releaseWithoutCleanup()
	if readErr != nil {
		return readErr
	}
	return releaseErr
}

func (owner *pidFileOwner) releaseWithoutCleanup() error {
	if owner == nil || owner.closed {
		return nil
	}
	owner.closed = true
	unlockErr := unlockFile(owner.file)
	closeErr := owner.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// inspectPIDFile 在同一个 fd 上探测锁并读取 PID。
func inspectPIDFile(path string) (bool, int, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	defer file.Close()

	locked, err := tryLockFile(file)
	if err != nil {
		return false, 0, err
	}
	if locked {
		defer unlockFile(file)
		pid, _, err := readPID(file)
		return false, pid, err
	}
	pid, exists, err := readPID(file)
	if err != nil {
		return false, 0, err
	}
	if !exists {
		return false, 0, errors.New("agent pid 文件被锁定但内容为空")
	}
	return true, pid, nil
}

func stopAgentProcess(pidFile string, ops processOperations, wait time.Duration) (int, error) {
	file, err := os.OpenFile(pidFile, os.O_RDWR, 0644)
	if err != nil {
		return 0, err
	}
	locked, err := tryLockFile(file)
	if err != nil {
		file.Close()
		return 0, err
	}
	if locked {
		clearErr := clearPID(file)
		unlockErr := unlockFile(file)
		closeErr := file.Close()
		if clearErr != nil {
			return 0, clearErr
		}
		if unlockErr != nil {
			return 0, unlockErr
		}
		if closeErr != nil {
			return 0, closeErr
		}
		return 0, errors.New("agent pid 文件未持有锁，服务未运行，已清理 stale PID")
	}

	pid, exists, err := readPID(file)
	closeErr := file.Close()
	if err != nil {
		return 0, err
	}
	if closeErr != nil {
		return 0, closeErr
	}
	if !exists {
		return 0, errors.New("active agent pid 文件内容为空")
	}
	if err := ops.terminate(pid); err != nil {
		return 0, err
	}

	attempts := int(wait / processPollInterval)
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		acquired, err := acquireReleasedPIDFile(pidFile, pid)
		if err != nil {
			return 0, err
		}
		if acquired {
			return pid, nil
		}
		ops.sleep(processPollInterval)
	}
	return 0, fmt.Errorf("等待 agent 服务进程 %d 释放 PID 锁超时", pid)
}

func acquireReleasedPIDFile(pidFile string, expectedPID int) (bool, error) {
	file, err := os.OpenFile(pidFile, os.O_RDWR, 0644)
	if err != nil {
		return false, err
	}
	locked, err := tryLockFile(file)
	if err != nil {
		file.Close()
		return false, err
	}
	if !locked {
		file.Close()
		return false, nil
	}
	defer unlockFile(file)
	defer file.Close()

	pid, exists, err := readPID(file)
	if err != nil {
		return false, err
	}
	if exists && pid != expectedPID {
		return false, fmt.Errorf("agent pid 文件已变更为 %d，拒绝清理", pid)
	}
	return true, clearPID(file)
}

type daemonReadinessProbes struct {
	ownership   func() (bool, int, error)
	tcpReady    func() bool
	childExited func() bool
	sleep       func(time.Duration)
}

func waitDaemonReady(childPID int, timeout time.Duration, probes daemonReadinessProbes) error {
	attempts := int(timeout / processPollInterval)
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		if probes.childExited() {
			return fmt.Errorf("agent daemon 子进程 %d 提前退出", childPID)
		}
		active, pid, err := probes.ownership()
		if err == nil && active && pid == childPID && probes.tcpReady() {
			return nil
		}
		probes.sleep(processPollInterval)
	}
	return fmt.Errorf("等待 agent daemon 子进程 %d 就绪超时", childPID)
}

func defaultDaemonReadinessProbes(pidFile, address string, childPID int) daemonReadinessProbes {
	dialAddress := address
	if strings.HasPrefix(dialAddress, ":") {
		dialAddress = "127.0.0.1" + dialAddress
	}
	return daemonReadinessProbes{
		ownership: func() (bool, int, error) {
			return inspectPIDFile(pidFile)
		},
		tcpReady: func() bool {
			conn, err := net.DialTimeout("tcp", dialAddress, processPollInterval)
			if err != nil {
				return false
			}
			conn.Close()
			return true
		},
		childExited: func() bool {
			return !util.CheckProcessExist(childPID)
		},
		sleep: time.Sleep,
	}
}
