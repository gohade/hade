package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanupPIDFileOnlyCleansCurrentPID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := cleanupPIDFile(pidFile, 43); err != nil {
		t.Fatal(err)
	}
	assertPIDFile(t, pidFile, "42")

	if err := cleanupPIDFile(pidFile, 42); err != nil {
		t.Fatal(err)
	}
	assertPIDFile(t, pidFile, "")
}

func TestStopRejectsMismatchedProcessWithoutKilling(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}
	killed := false
	ops := processOperations{
		command:   func(int) (string, error) { return "other worker", nil },
		terminate: func(int) error { killed = true; return nil },
		exists:    func(int) bool { return true },
		sleep:     func(time.Duration) {},
	}

	err := stopAgentProcess(pidFile, "hade", ops, time.Second)
	if err == nil || !strings.Contains(err.Error(), "身份不匹配") {
		t.Fatalf("got error %v", err)
	}
	if killed {
		t.Fatal("terminate must not be called")
	}
	assertPIDFile(t, pidFile, "42")
}

func TestStopWaitsForExitBeforeCleaningPID(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	if err := os.WriteFile(pidFile, []byte("42"), 0644); err != nil {
		t.Fatal(err)
	}
	checks := 0
	sawPIDWhileRunning := false
	ops := processOperations{
		command:   func(int) (string, error) { return "hade agent", nil },
		terminate: func(int) error { return nil },
		exists: func(int) bool {
			checks++
			if checks < 3 {
				content, _ := os.ReadFile(pidFile)
				sawPIDWhileRunning = sawPIDWhileRunning || string(content) == "42"
				return true
			}
			return false
		},
		sleep: func(time.Duration) {},
	}

	if err := stopAgentProcess(pidFile, "hade", ops, time.Second); err != nil {
		t.Fatal(err)
	}
	if !sawPIDWhileRunning || checks < 3 {
		t.Fatalf("did not wait for exit: checks=%d sawPID=%v", checks, sawPIDWhileRunning)
	}
	assertPIDFile(t, pidFile, "")
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
