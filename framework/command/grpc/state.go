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

// 获取启动的app的pid
var grpcStateCommand = &cobra.Command{
	Use:   "state",
	Short: "获取启动的grpc的pid",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// 获取pid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "grpc.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) > 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if util.CheckProcessExist(pid) {
				fmt.Println("grpc服务已经启动, pid:", pid)
				return nil
			}
		}
		fmt.Println("没有grpc服务存在")
		return nil
	},
}
