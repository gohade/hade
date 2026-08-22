package agent

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
)

var agentStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "停止一个已经启动的 agent 服务",
	RunE: func(c *cobra.Command, args []string) error {
		appService := c.GetContainer().MustMake(contract.AppKey).(contract.App)
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}
		if len(content) != 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if err := util.KillProcess(pid); err != nil {
				return err
			}
			if err := ioutil.WriteFile(serverPidFile, []byte{}, 0644); err != nil {
				return err
			}
			fmt.Println("停止 agent 服务进程:", pid)
		}
		return nil
	},
}
