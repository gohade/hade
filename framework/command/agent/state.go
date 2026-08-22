package agent

import (
	"fmt"
	"path/filepath"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
)

func newAgentStateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "state",
		Short: "获取启动的 agent 服务 pid",
		RunE: func(c *cobra.Command, args []string) error {
			appService := c.GetContainer().MustMake(contract.AppKey).(contract.App)
			serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")
			active, pid, err := inspectPIDFile(serverPidFile)
			if err != nil {
				return err
			}
			if !active {
				fmt.Println("没有agent服务存在")
				return nil
			}
			fmt.Println("agent服务已经启动, pid:", pid)
			return nil
		},
	}
}
