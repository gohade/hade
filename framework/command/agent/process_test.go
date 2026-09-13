//go:build (darwin || linux) && !pidlockstub

package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPIDFileOwnershipActiveAndStale(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.cleanup()

	active, pid, err := inspectPIDFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	if !active || pid != 42 {
		t.Fatalf("active=%v pid=%d", active, pid)
	}

	if err := owner.cleanup(); err != nil {
		t.Fatal(err)
	}
	assertPIDFile(t, pidFile, "")
	active, _, err = inspectPIDFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("unlocked pid file must be stale")
	}
}

func TestPIDFileSecondAcquireFails(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.cleanup()

	if _, err := acquirePIDFile(pidFile, 43); err == nil {
		t.Fatal("expected second acquire to fail")
	}
}

func TestPIDOwnerCleanupUsesSameLockAndChecksPID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePID(owner.file, 43); err != nil {
		t.Fatal(err)
	}
	if err := owner.cleanup(); err != nil {
		t.Fatal(err)
	}
	assertPIDFile(t, pidFile, "43")
}

func TestStopRejectsStalePIDWithoutTerminating(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")
	readyFile := filepath.Join(dir, "agent.ready")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeReadyFile(readyFile, 42); err != nil {
		t.Fatal(err)
	}
	terminated := false
	ops := processOperations{
		terminate: func(int) error { terminated = true; return nil },
		sleep:     func(time.Duration) {},
	}

	_, err := stopAgentLocked(pidFile, readyFile, ops, time.Second)
	if err == nil || !strings.Contains(err.Error(), "未持有锁") {
		t.Fatalf("got error %v", err)
	}
	if terminated {
		t.Fatal("terminate must not be called for stale pid file")
	}
	assertPIDFile(t, pidFile, "")
	if _, statErr := os.Stat(readyFile); !os.IsNotExist(statErr) {
		t.Fatalf("stale ready was not removed: %v", statErr)
	}
}

func TestStaleStopPreservesReadyForDifferentPID(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")
	readyFile := filepath.Join(dir, "agent.ready")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeReadyFile(readyFile, 43); err != nil {
		t.Fatal(err)
	}
	ops := processOperations{
		terminate: func(int) error { t.Fatal("must not terminate"); return nil },
		sleep:     func(time.Duration) {},
	}
	if _, err := stopAgentLocked(pidFile, readyFile, ops, time.Second); !errors.Is(err, ErrAgentNotRunning) {
		t.Fatalf("got error %v", err)
	}
	assertPIDFile(t, readyFile, "43")
}

func TestStopWaitsForLockReleaseBeforeCleaningPID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	sleepCount := 0
	ops := processOperations{
		terminate: func(int) error { return nil },
		sleep: func(time.Duration) {
			sleepCount++
			if sleepCount == 2 {
				if err := owner.releaseWithoutCleanup(); err != nil {
					t.Fatal(err)
				}
			}
		},
	}

	pid, err := stopAgentProcess(pidFile, ops, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if pid != 42 || sleepCount < 2 {
		t.Fatalf("pid=%d sleeps=%d", pid, sleepCount)
	}
	assertPIDFile(t, pidFile, "")
}

func TestStopTimeoutPreservesPID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.releaseWithoutCleanup()
	ops := processOperations{
		terminate: func(int) error { return nil },
		sleep:     func(time.Duration) {},
	}

	if _, err := stopAgentProcess(pidFile, ops, 250*time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
	assertPIDFile(t, pidFile, "42")
}

func TestStopTimeoutReleasesLifecycleBeforeNextStart(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")
	lockFile := filepath.Join(dir, "agent.lifecycle.lock")
	owner, err := acquirePIDFile(pidFile, 42)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.releaseWithoutCleanup()
	ops := processOperations{
		terminate: func(int) error { return nil },
		sleep:     func(time.Duration) {},
	}

	err = withLifecycleLock(lockFile, func(*exclusiveFileLock) error {
		_, err := stopAgentProcess(pidFile, ops, 250*time.Millisecond)
		return err
	})
	if err == nil {
		t.Fatal("expected stop timeout")
	}
	assertPIDFile(t, pidFile, "42")
	next, err := acquireLifecycleLock(lockFile)
	if err != nil {
		t.Fatalf("next start cannot enter: %v", err)
	}
	if err := next.release(); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleLockRejectsConcurrentEntry(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "agent.lifecycle.lock")
	first, err := acquireLifecycleLock(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	defer first.release()
	if _, err := acquireLifecycleLock(lockFile); err == nil {
		t.Fatal("expected concurrent lifecycle lock to fail")
	}
}

func TestDaemonParentHoldsLifecycleUntilReadyReturns(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "agent.lifecycle.lock")
	err := withLifecycleLock(lockFile, func(*exclusiveFileLock) error {
		probes := daemonReadinessProbes{
			ownership: func() (bool, int, error) {
				if concurrent, err := acquireLifecycleLock(lockFile); err == nil {
					concurrent.release()
					t.Fatal("concurrent lifecycle entry succeeded before ready")
				}
				return true, 42, nil
			},
			readyPID:    func() (int, bool, error) { return 42, true, nil },
			childExited: func() bool { return false },
			now:         advancingClock(),
			sleep:       func(time.Duration) {},
		}
		if err := waitDaemonReady(42, time.Second, probes); err != nil {
			return err
		}
		if concurrent, err := acquireLifecycleLock(lockFile); err == nil {
			concurrent.release()
			t.Fatal("concurrent lifecycle entry succeeded before parent completion")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	next, err := acquireLifecycleLock(lockFile)
	if err != nil {
		t.Fatalf("lifecycle remained locked: %v", err)
	}
	if err := next.release(); err != nil {
		t.Fatal(err)
	}
}

func TestDaemonAuthorizationRejectsForgedFDs(t *testing.T) {
	t.Run("invalid fd", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "agent.daemon.auth")
		if _, err := validateDaemonAuthorization(path, "999999"); err == nil {
			t.Fatal("invalid fd was accepted")
		}
	})

	t.Run("wrong inode", func(t *testing.T) {
		dir := t.TempDir()
		expected, err := acquireLifecycleLock(filepath.Join(dir, "expected.auth"))
		if err != nil {
			t.Fatal(err)
		}
		defer expected.release()
		other, err := acquireLifecycleLock(filepath.Join(dir, "other.auth"))
		if err != nil {
			t.Fatal(err)
		}
		defer other.release()
		fd, err := syscall.Dup(int(other.file.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		defer syscall.Close(fd)
		if _, err := validateDaemonAuthorization(expected.file.Name(), strconv.Itoa(fd)); err == nil {
			t.Fatal("wrong inode fd was accepted")
		}
		var stat syscall.Stat_t
		if err := syscall.Fstat(fd, &stat); err != nil {
			t.Fatalf("validation closed caller-owned fd: %v", err)
		}
	})

	t.Run("unlocked ordinary file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "agent.daemon.auth")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		fd, err := syscall.Dup(int(file.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		defer syscall.Close(fd)
		if _, err := validateDaemonAuthorization(path, strconv.Itoa(fd)); err == nil {
			t.Fatal("unlocked fd env bypassed lifecycle")
		}
		var stat syscall.Stat_t
		if err := syscall.Fstat(fd, &stat); err != nil {
			t.Fatalf("validation closed caller-owned fd: %v", err)
		}
	})
}

func TestDaemonAuthorizationAcceptsInheritedLockedSameInodeFD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.daemon.auth")
	owner, err := acquireLifecycleLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.release()
	fd, err := syscall.Dup(int(owner.file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)
	authorization, err := validateDaemonAuthorization(path, strconv.Itoa(fd))
	if err != nil {
		t.Fatal(err)
	}
	if err := authorization.discard(); err != nil {
		t.Fatal(err)
	}
}

func TestNonRebornProcessCannotUseValidAuthorizationFD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.daemon.auth")
	owner, err := acquireLifecycleLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.release()
	fd, err := syscall.Dup(int(owner.file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)

	authorization, err := resolveDaemonAuthorization(
		path,
		func(string) (string, bool) { return strconv.Itoa(fd), true },
		func() bool { return false },
	)
	if err == nil || authorization != nil {
		t.Fatalf("non-reborn authorization accepted: auth=%v err=%v", authorization, err)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		t.Fatalf("original fd was closed: %v", err)
	}
}

func TestAuthorizationDuplicateSurvivesOriginalFDReuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.daemon.auth")
	owner, err := acquireLifecycleLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.release()

	originalFD, err := syscall.Dup(int(owner.file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(os.DevNull)
	if err != nil {
		syscall.Close(originalFD)
		t.Fatal(err)
	}
	defer source.Close()

	authorization, err := validateDaemonAuthorization(path, strconv.Itoa(originalFD))
	if err != nil {
		syscall.Close(originalFD)
		t.Fatal(err)
	}
	if err := syscall.Close(originalFD); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Dup2(int(source.Fd()), originalFD); err != nil {
		t.Fatal(err)
	}
	reused := os.NewFile(uintptr(originalFD), "reused-original-fd")
	if reused == nil {
		t.Fatal("failed to wrap reused fd")
	}
	defer reused.Close()

	if err := authorization.discard(); err != nil {
		t.Fatal(err)
	}
	if _, err := reused.Stat(); err != nil {
		t.Fatalf("discard closed reused original fd: %v", err)
	}
}

func TestReadyCleanupOnlyCleansCurrentPID(t *testing.T) {
	readyFile := filepath.Join(t.TempDir(), "agent.ready")
	if err := writeReadyFile(readyFile, 42); err != nil {
		t.Fatal(err)
	}
	if err := cleanupReadyFile(readyFile, 43); err != nil {
		t.Fatal(err)
	}
	assertPIDFile(t, readyFile, "42")
	if err := cleanupReadyFile(readyFile, 42); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(readyFile); !os.IsNotExist(err) {
		t.Fatalf("ready file still exists: %v", err)
	}
}

func TestWaitDaemonReadyRequiresLockAndMatchingReadyPID(t *testing.T) {
	step := 0
	probes := daemonReadinessProbes{
		ownership: func() (bool, int, error) {
			step++
			switch step {
			case 1:
				return false, 0, nil
			case 2:
				return true, 41, nil
			default:
				return true, 42, nil
			}
		},
		readyPID: func() (int, bool, error) {
			if step < 3 {
				return 41, true, nil
			}
			return 42, true, nil
		},
		childExited: func() bool { return false },
		now:         advancingClock(),
		sleep:       func(time.Duration) {},
	}
	if err := waitDaemonReady(42, time.Second, probes); err != nil {
		t.Fatal(err)
	}

	t.Run("wrong ready pid times out", func(t *testing.T) {
		now := time.Unix(0, 0)
		probes := daemonReadinessProbes{
			ownership:   func() (bool, int, error) { return true, 42, nil },
			readyPID:    func() (int, bool, error) { return 41, true, nil },
			childExited: func() bool { return false },
			now:         func() time.Time { return now },
			sleep:       func(d time.Duration) { now = now.Add(d) },
		}
		if err := waitDaemonReady(42, 250*time.Millisecond, probes); err == nil {
			t.Fatal("wrong ready pid must not succeed")
		}
	})
}

func TestWaitDaemonReadyFailsForChildExitAndTimeout(t *testing.T) {
	t.Run("child exited", func(t *testing.T) {
		probes := daemonReadinessProbes{
			ownership:   func() (bool, int, error) { return false, 0, nil },
			readyPID:    func() (int, bool, error) { return 0, false, nil },
			childExited: func() bool { return true },
			now:         advancingClock(),
			sleep:       func(time.Duration) {},
		}
		if err := waitDaemonReady(42, time.Second, probes); err == nil {
			t.Fatal("expected child exit error")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		now := time.Unix(0, 0)
		probes := daemonReadinessProbes{
			ownership:   func() (bool, int, error) { return false, 0, nil },
			readyPID:    func() (int, bool, error) { return 0, false, nil },
			childExited: func() bool { return false },
			now:         func() time.Time { return now },
			sleep:       func(d time.Duration) { now = now.Add(d) },
		}
		if err := waitDaemonReady(42, 250*time.Millisecond, probes); err == nil {
			t.Fatal("expected timeout")
		}
	})
}

func TestWaitDaemonReadyRechecksChildAfterReadyMatches(t *testing.T) {
	checks := 0
	probes := daemonReadinessProbes{
		ownership: func() (bool, int, error) { return true, 42, nil },
		readyPID:  func() (int, bool, error) { return 42, true, nil },
		childExited: func() bool {
			checks++
			return checks == 2
		},
		now:   advancingClock(),
		sleep: func(time.Duration) {},
	}
	if err := waitDaemonReady(42, time.Second, probes); err == nil {
		t.Fatal("ready child exited before success")
	}
}

func TestAbortDaemonChildTerminatesKillsAndWaits(t *testing.T) {
	calls := make([]string, 0, 3)
	ops := daemonChildOperations{
		terminate: func() error { calls = append(calls, "terminate"); return nil },
		kill:      func() error { calls = append(calls, "kill"); return nil },
		wait:      func() error { calls = append(calls, "wait"); return nil },
	}
	if err := abortDaemonChild(ops); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls, ","); got != "terminate,kill,wait" {
		t.Fatalf("calls = %s", got)
	}
}

func TestAbortDaemonChildPropagatesErrorsAndStillWaits(t *testing.T) {
	waited := false
	ops := daemonChildOperations{
		terminate: func() error { return errors.New("terminate failed") },
		kill:      func() error { return nil },
		wait:      func() error { waited = true; return nil },
	}
	if err := abortDaemonChild(ops); err == nil || !strings.Contains(err.Error(), "terminate failed") {
		t.Fatalf("got error %v", err)
	}
	if !waited {
		t.Fatal("child was not reaped")
	}
}

func TestWaitDaemonFailureAbortsReapsAndCleans(t *testing.T) {
	for _, test := range []struct {
		name   string
		exited bool
	}{
		{name: "child exited", exited: true},
		{name: "timeout", exited: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Unix(0, 0)
			calls := make([]string, 0, 4)
			probes := daemonReadinessProbes{
				ownership:   func() (bool, int, error) { return false, 0, nil },
				readyPID:    func() (int, bool, error) { return 0, false, nil },
				childExited: func() bool { return test.exited },
				now:         func() time.Time { return now },
				sleep:       func(d time.Duration) { now = now.Add(d) },
			}
			err := waitDaemonOrAbort(
				42,
				250*time.Millisecond,
				probes,
				daemonChildOperations{
					terminate: func() error { calls = append(calls, "terminate"); return nil },
					kill:      func() error { calls = append(calls, "kill"); return nil },
					wait:      func() error { calls = append(calls, "wait"); return nil },
				},
				func() error { calls = append(calls, "cleanup"); return nil },
			)
			if err == nil {
				t.Fatal("expected readiness failure")
			}
			if got := strings.Join(calls, ","); got != "terminate,kill,wait,cleanup" {
				t.Fatalf("calls = %s", got)
			}
		})
	}
}

func TestRestartContinuesAfterStaleAgent(t *testing.T) {
	started := false
	err := restartAfterStop(
		func() error { return ErrAgentNotRunning },
		func() error { started = true; return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("start was not called")
	}
}

func TestCleanupErrorsAreCombinedWithMainError(t *testing.T) {
	err := finishAgentServe(
		errors.New("serve failed"),
		func() error { return errors.New("listener cleanup failed") },
		func() error { return errors.New("ready cleanup failed") },
		func() error { return errors.New("owner cleanup failed") },
	)
	for _, want := range []string{"serve failed", "listener cleanup failed", "ready cleanup failed", "owner cleanup failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
}

func TestLifecycleReleaseErrorIsCombinedWithMainError(t *testing.T) {
	lock := &exclusiveFileLock{
		releaseFn: func() error { return errors.New("release failed") },
	}
	err := runWithLifecycleLock(lock, func(*exclusiveFileLock) error {
		return errors.New("main failed")
	})
	if err == nil || !strings.Contains(err.Error(), "main failed") || !strings.Contains(err.Error(), "release failed") {
		t.Fatalf("got error %v", err)
	}
}

func advancingClock() func() time.Time {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(processPollInterval)
		return now
	}
}

func assertPIDFile(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); got != want {
		t.Fatalf("pid file = %q, want %q", got, want)
	}
}
