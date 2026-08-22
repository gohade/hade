//go:build darwin || linux

package agent

import (
	"os"
	"path/filepath"
	"strings"
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
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}
	terminated := false
	ops := processOperations{
		terminate: func(int) error { terminated = true; return nil },
		sleep:     func(time.Duration) {},
	}

	_, err := stopAgentProcess(pidFile, ops, time.Second)
	if err == nil || !strings.Contains(err.Error(), "未持有锁") {
		t.Fatalf("got error %v", err)
	}
	if terminated {
		t.Fatal("terminate must not be called for stale pid file")
	}
	assertPIDFile(t, pidFile, "")
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

func TestWaitDaemonReadyRequiresLockPIDAndTCP(t *testing.T) {
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
		tcpReady:    func() bool { return step >= 3 },
		childExited: func() bool { return false },
		sleep:       func(time.Duration) {},
	}
	if err := waitDaemonReady(42, time.Second, probes); err != nil {
		t.Fatal(err)
	}
}

func TestWaitDaemonReadyFailsForChildExitAndTimeout(t *testing.T) {
	t.Run("child exited", func(t *testing.T) {
		probes := daemonReadinessProbes{
			ownership:   func() (bool, int, error) { return false, 0, nil },
			tcpReady:    func() bool { return false },
			childExited: func() bool { return true },
			sleep:       func(time.Duration) {},
		}
		if err := waitDaemonReady(42, time.Second, probes); err == nil {
			t.Fatal("expected child exit error")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		probes := daemonReadinessProbes{
			ownership:   func() (bool, int, error) { return false, 0, nil },
			tcpReady:    func() bool { return false },
			childExited: func() bool { return false },
			sleep:       func(time.Duration) {},
		}
		if err := waitDaemonReady(42, 250*time.Millisecond, probes); err == nil {
			t.Fatal("expected timeout")
		}
	})
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
