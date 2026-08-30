package grpc

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

// 重新启动一个app服务
var grpcRestartCommand = &cobra.Command{
	Use:   "restart",
	Short: "重新启动一个grpc服务",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "grpc.pid")

		if !util.Exists(serverPidFile) {
			appDaemon = true
			// 直接daemon方式启动apps
			return grpcStartCommand.RunE(c, args)
		}

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) != 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if util.CheckProcessExist(pid) {
				// 杀死进程
				if err := util.KillProcess(pid); err != nil {
					return err
				}

				// 获取closeWait
				closeWait := 5
				configService := container.MustMake(contract.ConfigKey).(contract.Config)
				if configService.IsExist("grpc.close_wait") {
					closeWait = configService.GetInt("grpc.close_wait")
				}

				// 确认进程已经关闭,每秒检测一次， 最多检测closeWait * 2秒
				for i := 0; i < closeWait*2; i++ {
					if util.CheckProcessExist(pid) == false {
						break
					}
					time.Sleep(1 * time.Second)
				}

				// 如果进程等待了2*closeWait之后还没结束，返回错误，不进行后续的操作
				if util.CheckProcessExist(pid) == true {
					fmt.Println("结束进程失败:"+strconv.Itoa(pid), "请查看原因")
					return errors.New("结束进程失败")
				}
				if err := ioutil.WriteFile(serverPidFile, []byte{}, 0644); err != nil {
					return err
				}

				fmt.Println("结束进程成功:" + strconv.Itoa(pid))
			}
		}

		appDaemon = true
		// 直接daemon方式启动apps
		return grpcStartCommand.RunE(c, args)
	},
}
