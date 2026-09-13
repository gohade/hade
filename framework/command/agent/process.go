package agent

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gohade/hade/framework/util"
	"github.com/sevlyar/go-daemon"
)

const processPollInterval = 100 * time.Millisecond

var ErrAgentNotRunning = errors.New("agent 服务未运行")

type processOperations struct {
	terminate func(int) error
	sleep     func(time.Duration)
	now       func() time.Time
	wasReborn func() bool
}

func defaultProcessOperations() processOperations {
	return processOperations{
		terminate: util.KillProcess,
		sleep:     time.Sleep,
		now:       time.Now,
		wasReborn: daemon.WasReborn,
	}
}

type exclusiveFileLock struct {
	file      *os.File
	released  bool
	releaseFn func() error
}

type daemonAuthorization struct {
	file *os.File
}

func validateDaemonAuthorization(path, fdText string) (*daemonAuthorization, error) {
	fd, err := strconv.Atoi(fdText)
	if err != nil || fd < 3 {
		return nil, fmt.Errorf("无效的 daemon authorization fd %q", fdText)
	}
	duplicate, err := duplicateFD(fd)
	if err != nil {
		return nil, fmt.Errorf("复制 daemon authorization fd %d 失败: %w", fd, err)
	}
	file := os.NewFile(uintptr(duplicate), "agent-daemon-authorization-duplicate")
	if file == nil {
		closeFD(duplicate)
		return nil, fmt.Errorf("daemon authorization fd %d 无效", fd)
	}
	fail := func(err error) (*daemonAuthorization, error) {
		return nil, mergeErrors(err, file.Close())
	}

	fdInfo, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	pathInfo, err := os.Stat(path)
	if err != nil {
		return fail(err)
	}
	if !os.SameFile(fdInfo, pathInfo) {
		return fail(errors.New("daemon authorization fd 与授权文件 inode 不匹配"))
	}

	probe, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return fail(err)
	}
	probeLocked, probeErr := tryLockFile(probe)
	if probeErr != nil {
		return fail(mergeErrors(probeErr, probe.Close()))
	}
	if probeLocked {
		return fail(mergeErrors(
			errors.New("daemon authorization 文件未持有继承锁"),
			unlockFile(probe),
			probe.Close(),
		))
	}
	if err := probe.Close(); err != nil {
		return fail(err)
	}

	inheritedLocked, err := tryLockFile(file)
	if err != nil {
		return fail(err)
	}
	if !inheritedLocked {
		return fail(errors.New("daemon authorization fd 未继承同一锁描述"))
	}
	return &daemonAuthorization{file: file}, nil
}

func resolveDaemonAuthorization(
	path string,
	lookupEnv func(string) (string, bool),
	wasReborn func() bool,
) (*daemonAuthorization, error) {
	fdText, hasFD := lookupEnv(agentDaemonAuthFDEnv)
	reborn := wasReborn()
	if hasFD && !reborn {
		return nil, errors.New("非 reborn 进程不得使用 daemon authorization fd")
	}
	if reborn && !hasFD {
		return nil, errors.New("daemon child 缺少继承 authorization fd")
	}
	if !hasFD {
		return nil, nil
	}
	return validateDaemonAuthorization(path, fdText)
}

func (authorization *daemonAuthorization) discard() error {
	if authorization == nil || authorization.file == nil {
		return nil
	}
	err := authorization.file.Close()
	authorization.file = nil
	return err
}

func acquireLifecycleLock(path string) (*exclusiveFileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	locked, err := tryLockFile(file)
	if err != nil {
		return nil, mergeErrors(err, file.Close())
	}
	if !locked {
		return nil, mergeErrors(errors.New("agent lifecycle 正由其他命令持有"), file.Close())
	}
	return &exclusiveFileLock{file: file}, nil
}

func (lock *exclusiveFileLock) release() error {
	if lock == nil || lock.released {
		return nil
	}
	lock.released = true
	if lock.releaseFn != nil {
		return lock.releaseFn()
	}
	unlockErr := unlockFile(lock.file)
	closeErr := lock.file.Close()
	return mergeErrors(unlockErr, closeErr)
}

func withLifecycleLock(path string, fn func(*exclusiveFileLock) error) error {
	lock, err := acquireLifecycleLock(path)
	if err != nil {
		return err
	}
	return runWithLifecycleLock(lock, fn)
}

func runWithLifecycleLock(lock *exclusiveFileLock, fn func(*exclusiveFileLock) error) error {
	mainErr := fn(lock)
	return mergeErrors(mainErr, lock.release())
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
		return nil, mergeErrors(err, file.Close())
	}
	if !locked {
		return nil, mergeErrors(fmt.Errorf("agent pid 文件已被其他进程锁定"), file.Close())
	}
	if err := writePID(file, pid); err != nil {
		return nil, mergeErrors(err, unlockFile(file), file.Close())
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

func writeReadyFile(path string, pid int) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if _, err := file.WriteString(strconv.Itoa(pid)); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func readReadyPID(path string) (int, bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return 0, false, nil
	}
	pid, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("无效的 agent ready pid %q: %w", value, err)
	}
	return pid, true, nil
}

func cleanupReadyFile(path string, currentPID int) error {
	pid, exists, err := readReadyPID(path)
	if err != nil || !exists || pid != currentPID {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
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
	return mergeErrors(readErr, releaseErr)
}

func (owner *pidFileOwner) releaseWithoutCleanup() error {
	if owner == nil || owner.closed {
		return nil
	}
	owner.closed = true
	unlockErr := unlockFile(owner.file)
	closeErr := owner.file.Close()
	return mergeErrors(unlockErr, closeErr)
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
	if os.IsNotExist(err) {
		return 0, ErrAgentNotRunning
	}
	if err != nil {
		return 0, err
	}
	locked, err := tryLockFile(file)
	if err != nil {
		file.Close()
		return 0, err
	}
	if locked {
		stalePID, _, readErr := readPID(file)
		clearErr := clearPID(file)
		unlockErr := unlockFile(file)
		closeErr := file.Close()
		if cleanupErr := mergeErrors(readErr, clearErr, unlockErr, closeErr); cleanupErr != nil {
			return stalePID, cleanupErr
		}
		return stalePID, fmt.Errorf("%w：pid 文件未持有锁，已清理 stale PID", ErrAgentNotRunning)
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

	now := ops.now
	if now == nil {
		now = time.Now
	}
	deadline := now().Add(wait)
	for {
		acquired, err := acquireReleasedPIDFile(pidFile, pid)
		if err != nil {
			return 0, err
		}
		if acquired {
			return pid, nil
		}
		if !now().Before(deadline) {
			return 0, fmt.Errorf("等待 agent 服务进程 %d 释放 PID 锁超时", pid)
		}
		ops.sleep(processPollInterval)
	}
}

func acquireReleasedPIDFile(pidFile string, expectedPID int) (bool, error) {
	file, err := os.OpenFile(pidFile, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return true, nil
	}
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

	pid, exists, err := readPID(file)
	if err != nil {
		return false, mergeErrors(err, unlockFile(file), file.Close())
	}
	if exists && pid != expectedPID {
		return false, mergeErrors(
			fmt.Errorf("agent pid 文件已变更为 %d，拒绝清理", pid),
			unlockFile(file),
			file.Close(),
		)
	}
	clearErr := clearPID(file)
	return true, mergeErrors(clearErr, unlockFile(file), file.Close())
}

type daemonReadinessProbes struct {
	ownership   func() (bool, int, error)
	readyPID    func() (int, bool, error)
	childExited func() bool
	sleep       func(time.Duration)
	now         func() time.Time
}

func waitDaemonReady(childPID int, timeout time.Duration, probes daemonReadinessProbes) error {
	now := probes.now
	if now == nil {
		now = time.Now
	}
	deadline := now().Add(timeout)
	for {
		if probes.childExited() {
			return fmt.Errorf("agent daemon 子进程 %d 提前退出", childPID)
		}
		active, pid, err := probes.ownership()
		if err == nil && active && pid == childPID {
			readyPID, ready, readyErr := probes.readyPID()
			if readyErr == nil && ready && readyPID == childPID {
				if probes.childExited() {
					return fmt.Errorf("agent daemon 子进程 %d 在就绪确认时退出", childPID)
				}
				return nil
			}
		}
		if !now().Before(deadline) {
			return fmt.Errorf("等待 agent daemon 子进程 %d 就绪超时", childPID)
		}
		probes.sleep(processPollInterval)
	}
}

func defaultDaemonReadinessProbes(pidFile, readyFile string, childPID int) daemonReadinessProbes {
	return daemonReadinessProbes{
		ownership: func() (bool, int, error) {
			return inspectPIDFile(pidFile)
		},
		readyPID: func() (int, bool, error) {
			return readReadyPID(readyFile)
		},
		childExited: func() bool {
			return !util.CheckProcessExist(childPID)
		},
		sleep: time.Sleep,
		now:   time.Now,
	}
}

type daemonChildOperations struct {
	terminate func() error
	kill      func() error
	wait      func() error
}

func abortDaemonChild(ops daemonChildOperations) error {
	var failures []string
	if err := ops.terminate(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		failures = append(failures, err.Error())
	}
	if err := ops.kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		failures = append(failures, err.Error())
	}
	if err := ops.wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		failures = append(failures, err.Error())
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func waitDaemonOrAbort(
	childPID int,
	timeout time.Duration,
	probes daemonReadinessProbes,
	childOps daemonChildOperations,
	cleanup func() error,
) error {
	readyErr := waitDaemonReady(childPID, timeout, probes)
	if readyErr == nil {
		return nil
	}
	return combineErrors(readyErr, abortDaemonChild(childOps), cleanup())
}

func restartAfterStop(stop, start func() error) error {
	if err := stop(); err != nil && !errors.Is(err, ErrAgentNotRunning) {
		return err
	}
	return start()
}

func cleanupFailedDaemonFiles(pidFile, readyFile, authFile string, childPID int) error {
	var failures []string
	if err := cleanupReadyFile(readyFile, childPID); err != nil {
		failures = append(failures, err.Error())
	}
	acquired, err := acquireReleasedPIDFile(pidFile, childPID)
	if err != nil {
		failures = append(failures, err.Error())
	} else if !acquired {
		failures = append(failures, "daemon 子进程退出后 PID 锁仍未释放")
	}
	if err := os.Remove(authFile); err != nil && !os.IsNotExist(err) {
		failures = append(failures, err.Error())
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func combineErrors(primary error, others ...error) error {
	return mergeErrors(append([]error{primary}, others...)...)
}

type combinedError struct {
	primary error
	message string
}

func (err *combinedError) Error() string {
	return err.message
}

func (err *combinedError) Unwrap() error {
	return err.primary
}

func mergeErrors(errs ...error) error {
	var failures []string
	var primary error
	for _, err := range errs {
		if err != nil {
			if primary == nil {
				primary = err
			}
			failures = append(failures, err.Error())
		}
	}
	if len(failures) == 0 {
		return nil
	}
	if len(failures) == 1 {
		return primary
	}
	return &combinedError{primary: primary, message: strings.Join(failures, "; ")}
}

func finishAgentServe(
	mainErr error,
	closeListener func() error,
	cleanupReady func() error,
	cleanupOwner func() error,
) error {
	return mergeErrors(mainErr, closeListener(), cleanupReady(), cleanupOwner())
}
