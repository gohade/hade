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

var agentStateCommand = &cobra.Command{
	Use:   "state",
	Short: "获取启动的 agent 服务 pid",
	RunE: func(c *cobra.Command, args []string) error {
		appService := c.GetContainer().MustMake(contract.AppKey).(contract.App)
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}
		if len(content) > 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if util.CheckProcessExist(pid) {
				fmt.Println("agent服务已经启动, pid:", pid)
				return nil
			}
		}
		fmt.Println("没有agent服务存在")
		return nil
	},
}
