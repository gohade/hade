package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"
	"hade/framework/util"

	"github.com/erikdubbelboer/gspt"
	"github.com/sevlyar/go-daemon"
)

var cronDeamon = false

func initCronCommand() *cobra.Command {

	cronStartCommand.Flags().BoolVarP(&cronDeamon, "deamon", "d", false, "start serve deamon")
	cronCommand.AddCommand(cronRestartCommand)
	cronCommand.AddCommand(cronStateCommand)
	cronCommand.AddCommand(cronStopCommand)
	cronCommand.AddCommand(cronListCommand)
	cronCommand.AddCommand(cronStartCommand)

	return cronCommand
}

var cronCommand = &cobra.Command{
	Use:   "cron",
	Short: "about cron command",
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
	Short: "list all cron command",
	RunE: func(c *cobra.Command, args []string) error {

		cronSpecs := c.Root().CronSepcs
		ps := [][]string{}
		for _, cronSpec := range cronSpecs {
			line := []string{cronSpec.Spec, cronSpec.Cmd.Use, cronSpec.Cmd.Short}
			ps = append(ps, line)
		}
		util.PrettyPrint(ps)

		return nil
	},
}

var cronStartCommand = &cobra.Command{
	Use:   "start",
	Short: "start cron command",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		pidFolder := appService.PidPath()
		serverPidFile := filepath.Join(pidFolder, "cron.pid")
		logFolder := appService.LogPath()
		serverLogFile := filepath.Join(logFolder, "cron.log")
		currentFolder := util.GetExecDirectory()
		// deamon mode
		if cronDeamon {
			cntxt := &daemon.Context{
				PidFileName: serverPidFile,
				PidFilePerm: 0664,
				LogFileName: serverLogFile,
				LogFilePerm: 0640,
				WorkDir:     currentFolder,
				Umask:       027,
				Args:        []string{"", "cron", "start", "--deamon=true"},
			}
			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				fmt.Println("cron serve started")
				fmt.Println("log file:", serverLogFile)
				return nil
			}
			defer cntxt.Release()
			fmt.Println("deamon started")
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
	Short: "restart cron command",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "cron.pid")

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
				if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
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

		cronDeamon = true
		return cronStartCommand.RunE(c, args)
	},
}

var cronStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "stop cron command",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "cron.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) > 0 {
			pid, err := strconv.Atoi(string(content))
			if err != nil {
				return err
			}
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
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
	Short: "cron serve state",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "cron.pid")

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
