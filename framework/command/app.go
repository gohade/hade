package command

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

var appDeamon = false
var appAddress = ""

func initAppCommand() *cobra.Command {

	appStartCommand.Flags().BoolVarP(&appDeamon, "deamon", "d", false, "start app deamon")
	appStartCommand.Flags().StringVar(&appAddress, "address", "", "set app address")
	appCommand.AddCommand(appStartCommand)
	appCommand.AddCommand(appRestartCommand)
	appCommand.AddCommand(appStateCommand)
	appCommand.AddCommand(appStopCommand)

	return appCommand
}

var appCommand = &cobra.Command{
	Use:   "app",
	Short: "start app serve",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

// appCommand start a app app
var appStartCommand = &cobra.Command{
	Use:   "start",
	Short: "start app server",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)
		configService := container.MustMake(contract.ConfigKey).(contract.Config)
		kernelService := container.MustMake(contract.KernelKey).(contract.Kernel)

		appURL := configService.GetString("app.url")
		if appAddress != "" {
			appURL = appAddress
		}

		if appURL == "" {
			appURL = "http://localhost:8080"
		}

		r := kernelService.HttpEngine()

		hadeURL, err := url.Parse(appURL)
		if err != nil {
			return err
		}
		hadeServer := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", hadeURL.Hostname(), hadeURL.Port()),
			Handler: r,
		}

		pidFolder := appService.PidPath()
		if !util.Exists(pidFolder) {
			os.MkdirAll(pidFolder, os.ModePerm)
		}
		serverPidFile := filepath.Join(pidFolder, "app.pid")
		logFolder := appService.LogPath()
		if !util.Exists(logFolder) {
			os.MkdirAll(logFolder, os.ModePerm)
		}
		// 应用日志
		serverLogFile := filepath.Join(logFolder, "app.log")
		currentFolder := util.GetExecDirectory()
		// deamon mode
		if appDeamon {
			cntxt := &daemon.Context{
				PidFileName: serverPidFile,
				PidFilePerm: 0664,
				LogFileName: serverLogFile,
				LogFilePerm: 0640,
				WorkDir:     currentFolder,
				Umask:       027,
				Args:        []string{"", "app", "start", "--deamon=true"},
			}
			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				fmt.Println("app serve started")
				fmt.Println("log file:", serverLogFile)
				return nil
			}
			defer cntxt.Release()
			fmt.Println("deamon started")
			gspt.SetProcTitle("hade app")
			err = hadeServer.ListenAndServe()
			if err != nil {
				fmt.Println(err)
			}
			return nil
		}

		// 记录app.pid
		content := strconv.Itoa(os.Getpid())
		fmt.Println("[PID]", content)
		err = ioutil.WriteFile(serverPidFile, []byte(content), 0644)
		if err != nil {
			return err
		}
		gspt.SetProcTitle("hade app")

		fmt.Println("app serve url:", appURL)
		err = hadeServer.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}

		return nil
	},
}

var appRestartCommand = &cobra.Command{
	Use:   "restart",
	Short: "restart app server",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "app.pid")

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
				if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
					return err
				}
				if err := ioutil.WriteFile(serverPidFile, []byte{}, 0644); err != nil {
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

		appDeamon = true
		return appStartCommand.RunE(c, args)
	},
}

var appStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "stop app server",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "app.pid")

		content, err := ioutil.ReadFile(serverPidFile)
		if err != nil {
			return err
		}

		if content != nil && len(content) != 0 {
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

var appStateCommand = &cobra.Command{
	Use:   "state",
	Short: "get app pid",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		// GetPid
		serverPidFile := filepath.Join(appService.PidPath(), "app.pid")

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
				fmt.Println("app server started, pid:", pid)
				return nil
			}
		}
		fmt.Println("no app server start")
		return nil
	},
}
