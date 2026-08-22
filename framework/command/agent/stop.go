package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
)

func agentStopWait(c *cobra.Command) time.Duration {
	wait := 5 * time.Second
	configService := c.GetContainer().MustMake(contract.ConfigKey).(contract.Config)
	if configService.IsExist("agent.close_wait") {
		wait = time.Duration(configService.GetInt("agent.close_wait")) * time.Second
	}
	return wait
}

func newAgentStopCommand(deps agentDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止一个已经启动的 agent 服务",
		RunE: func(c *cobra.Command, args []string) error {
			appService := c.GetContainer().MustMake(contract.AppKey).(contract.App)
			runtimeFolder := appService.RuntimeFolder()
			return withLifecycleLock(filepath.Join(runtimeFolder, "agent.lifecycle.lock"), func(*exclusiveFileLock) error {
				pid, err := stopAgentLocked(
					filepath.Join(runtimeFolder, "agent.pid"),
					filepath.Join(runtimeFolder, "agent.ready"),
					deps.process,
					agentStopWait(c),
				)
				if err != nil {
					return err
				}
				fmt.Println("停止 agent 服务进程:", pid)
				return nil
			})
		},
	}
}

func stopAgentLocked(pidFile, readyFile string, ops processOperations, wait time.Duration) (int, error) {
	pid, err := stopAgentProcess(pidFile, ops, wait)
	if err != nil {
		if errors.Is(err, ErrAgentNotRunning) {
			return pid, mergeErrors(err, cleanupReadyFile(readyFile, pid))
		}
		return 0, err
	}
	if err := cleanupReadyFile(readyFile, pid); err != nil {
		return 0, err
	}
	return pid, nil
}
