package agent

import (
	"os"
	"path/filepath"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
)

func newAgentRestartCommand(options *agentOptions, stopCommand, startCommand *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重新启动一个 agent 服务",
		RunE: func(c *cobra.Command, args []string) error {
			appService := c.GetContainer().MustMake(contract.AppKey).(contract.App)
			serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")
			if _, err := os.Stat(serverPidFile); err == nil {
				if err := stopCommand.RunE(c, args); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}

			options.daemon = true
			return startCommand.RunE(c, args)
		},
	}
}
