package grpc

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
)

// 停止一个已经启动的app服务
var grpcStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "停止一个已经启动的grpc服务",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "grpc.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) != 0 {
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
			fmt.Println("停止进程:", pid)
		}
		return nil
	},
}
