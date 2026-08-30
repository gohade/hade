package command

import (
	"fmt"
	"github.com/erikdubbelboer/gspt"
	"github.com/sevlyar/go-daemon"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
)

var cronDaemon = false

func initCronCommand() *cobra.Command {

	cronStartCommand.Flags().BoolVarP(&cronDaemon, "daemon", "d", false, "start serve daemon")
	cronCommand.AddCommand(cronRestartCommand)
	cronCommand.AddCommand(cronStateCommand)
	cronCommand.AddCommand(cronStopCommand)
	cronCommand.AddCommand(cronListCommand)
	cronCommand.AddCommand(cronStartCommand)

	return cronCommand
}

var cronCommand = &cobra.Command{
	Use:   "cron",
	Short: "定时任务相关命令",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

// serveCommand start a app serve
var cronListCommand = &cobra.Command{
	Use:   "list",
	Short: "列出所有的定时任务",
	RunE: func(c *cobra.Command, args []string) error {

		cronSpecs := c.Root().CronSpecs
		ps := [][]string{}
		for _, cronSpec := range cronSpecs {
			line := []string{cronSpec.Type, cronSpec.Spec, cronSpec.Cmd.Use, cronSpec.Cmd.Short, cronSpec.ServiceName}
			ps = append(ps, line)
		}
		util.PrettyPrint(ps)

		return nil
	},
}

// cron进程的启动服务
var cronStartCommand = &cobra.Command{
	Use:   "start",
	Short: "启动cron常驻进程",
	RunE: func(c *cobra.Command, args []string) error {
		// 获取容器
		container := c.GetContainer()
		// 获取容器中的app服务
		appService := container.MustMake(contract.AppKey).(contract.App)

		// 设置cron的日志地址和进程id地址
		pidFolder := appService.RuntimeFolder()
		serverPidFile := filepath.Join(pidFolder, "cron.pid")
		logFolder := appService.LogFolder()
		serverLogFile := filepath.Join(logFolder, "cron.log")
		currentFolder := appService.BaseFolder()

		// deamon 模式
		if cronDaemon {
			// 创建一个Context
			cntxt := &daemon.Context{
				// 设置pid文件
				PidFileName: serverPidFile,
				PidFilePerm: 0664,
				// 设置日志文件
				LogFileName: serverLogFile,
				LogFilePerm: 0640,
				// 设置工作路径
				WorkDir: currentFolder,
				// 设置所有设置文件的mask，默认为750
				Umask: 027,
				// 子进程的参数，按照这个参数设置，子进程的命令为 ./hade cron start --daemon=true
				Args: []string{"", "cron", "start", "--daemon=true"},
			}
			// 启动子进程，d不为空表示当前是父进程，d为空表示当前是子进程
			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				// 父进程直接打印启动成功信息，不做任何操作
				fmt.Println("cron serve started, pid:", d.Pid)
				fmt.Println("log file:", serverLogFile)
				return nil
			}

			// 子进程执行Cron.Run
			defer cntxt.Release()
			fmt.Println("daemon started")
			gspt.SetProcTitle("hade cron")
			c.Root().Cron.Run()
			return nil
		}

		// not deamon mode
		fmt.Println("start cron job")
		content := strconv.Itoa(os.Getpid())
		fmt.Println("[PID]", content)
		err := ioutil.WriteFile(serverPidFile, []byte(content), 0664)
		if err != nil {
			return err
		}

		gspt.SetProcTitle("hade cron")
		c.Root().Cron.Run()
		return nil
	},
}

var cronRestartCommand = &cobra.Command{
	Use:   "restart",
	Short: "重启cron常驻进程",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "cron.pid")

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
				if err := util.KillProcess(pid); err != nil {
					return err
				}
				// check process closed
				for i := 0; i < 10; i++ {
					if util.CheckProcessExist(pid) == false {
						break
					}
					time.Sleep(1 * time.Second)
				}
				fmt.Println("kill process:" + strconv.Itoa(pid))
			}
		}

		cronDaemon = true
		return cronStartCommand.RunE(c, args)
	},
}

var cronStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "停止cron常驻进程",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "cron.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) > 0 {
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
			fmt.Println("stop pid:", pid)
		}
		return nil
	},
}

var cronStateCommand = &cobra.Command{
	Use:   "state",
	Short: "cron常驻进程状态",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.RuntimeFolder(), "cron.pid")

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
				fmt.Println("cron server started, pid:", pid)
				return nil
			}
		}
		fmt.Println("no cron server start")
		return nil
	},
}
