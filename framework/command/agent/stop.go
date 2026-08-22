package agent

import (
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
			serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")
			pid, err := stopAgentProcess(serverPidFile, deps.process, agentStopWait(c))
			if err != nil {
				return err
			}
			fmt.Println("停止 agent 服务进程:", pid)
			return nil
		},
	}
}
