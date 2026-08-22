package agent

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
	"github.com/pkg/errors"
)

var agentRestartCommand = &cobra.Command{
	Use:   "restart",
	Short: "重新启动一个 agent 服务",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "agent.pid")

		if !util.Exists(serverPidFile) {
			agentDaemon = true
			return agentStartCommand.RunE(c, args)
		}

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}
		if len(content) != 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if util.CheckProcessExist(pid) {
				if err := util.KillProcess(pid); err != nil {
					return err
				}

				closeWait := 5
				configService := container.MustMake(contract.ConfigKey).(contract.Config)
				if configService.IsExist("agent.close_wait") {
					closeWait = configService.GetInt("agent.close_wait")
				}
				for i := 0; i < closeWait*2; i++ {
					if !util.CheckProcessExist(pid) {
						break
					}
					time.Sleep(time.Second)
				}
				if util.CheckProcessExist(pid) {
					fmt.Println("结束进程失败:"+strconv.Itoa(pid), "请查看原因")
					return errors.New("结束进程失败")
				}
				if err := ioutil.WriteFile(serverPidFile, []byte{}, 0644); err != nil {
					return err
				}
				fmt.Println("结束 agent 服务进程成功:" + strconv.Itoa(pid))
			}
		}

		agentDaemon = true
		return agentStartCommand.RunE(c, args)
	},
}
