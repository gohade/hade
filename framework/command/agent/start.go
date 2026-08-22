package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/erikdubbelboer/gspt"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
	"github.com/sevlyar/go-daemon"
)

func normalizeAgentAddress(address string) string {
	if address == "" {
		return ":8889"
	}
	if !strings.Contains(address, ":") {
		return ":" + address
	}
	return address
}

func resolveAgentAddress(flagAddress, envAddress, configPort string, configExists bool) string {
	if flagAddress != "" {
		return flagAddress
	}
	if envAddress != "" {
		return envAddress
	}
	if configExists {
		return normalizeAgentAddress(configPort)
	}
	return ":8889"
}

func startAgentServe(server *http.Server, c framework.Container) error {
	go func() {
		_ = server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(quit)
	<-quit

	closeWait := 5
	configService := c.MustMake(contract.ConfigKey).(contract.Config)
	if configService.IsExist("agent.close_wait") {
		closeWait = configService.GetInt("agent.close_wait")
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(closeWait)*time.Second)
	defer cancel()
	return server.Shutdown(timeoutCtx)
}

var agentStartCommand = &cobra.Command{
	Use:   "start",
	Short: "启动一个 agent 服务",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		kernelService := container.MustMake(contract.KernelKey).(contract.Kernel)
		core := kernelService.AgentEngine()

		envService := container.MustMake(contract.EnvKey).(contract.Env)
		configService := container.MustMake(contract.ConfigKey).(contract.Config)
		configExists := configService.IsExist("agent.port")
		configPort := ""
		if configExists {
			configPort = configService.GetString("agent.port")
		}
		agentAddress = resolveAgentAddress(agentAddress, envService.Get("AGENT_ADDRESS"), configPort, configExists)

		server := &http.Server{Handler: core, Addr: agentAddress}
		appService := container.MustMake(contract.AppKey).(contract.App)

		pidFolder := appService.RuntimeFolder()
		if !util.Exists(pidFolder) {
			if err := os.MkdirAll(pidFolder, os.ModePerm); err != nil {
				return err
			}
		}
		serverPidFile := filepath.Join(pidFolder, "agent.pid")

		logFolder := appService.LogFolder()
		if !util.Exists(logFolder) {
			if err := os.MkdirAll(logFolder, os.ModePerm); err != nil {
				return err
			}
		}
		serverLogFile := filepath.Join(logFolder, "agent.log")

		processName := "hade agent"
		if len(os.Args) > 0 {
			processName = filepath.Base(os.Args[0]) + " agent"
		}

		if agentDaemon {
			parentArgs := make([]string, 0, len(os.Args))
			for _, arg := range os.Args {
				if strings.HasPrefix(arg, "--") {
					if strings.Contains(arg, "--daemon=") {
						continue
					}
					parentArgs = append(parentArgs, arg)
				}
			}
			subArgs := []string{filepath.Base(os.Args[0]), "agent", "start", "--daemon=true"}
			subArgs = append(subArgs, parentArgs...)
			cntxt := &daemon.Context{
				PidFileName: serverPidFile,
				PidFilePerm: 0664,
				LogFileName: serverLogFile,
				LogFilePerm: 0640,
				WorkDir:     util.GetExecDirectory(),
				Umask:       027,
				Args:        subArgs,
				Env:         os.Environ(),
			}
			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				printAgentStarted(processName, strconv.Itoa(d.Pid), agentAddress, appService)
				return nil
			}
			defer cntxt.Release()
			fmt.Println("daemon started")
			gspt.SetProcTitle(processName)
			return startAgentServe(server, container)
		}

		content := strconv.Itoa(os.Getpid())
		if err := ioutil.WriteFile(serverPidFile, []byte(content), 0644); err != nil {
			return err
		}
		gspt.SetProcTitle(processName)
		printAgentStarted(processName, content, agentAddress, appService)
		return startAgentServe(server, container)
	},
}

func printAgentStarted(processName, pid, address string, appService contract.App) {
	fmt.Println("成功启动进程:", processName)
	fmt.Println("进程pid:", pid)
	showAddress := address
	if strings.HasPrefix(address, ":") {
		showAddress = "http://localhost" + address
	}
	fmt.Println("监听地址:", showAddress)
	fmt.Println("基础路径:", appService.BaseFolder())
	fmt.Println("日志路径:", appService.LogFolder())
	fmt.Println("运行路径:", appService.RuntimeFolder())
	fmt.Println("配置路径:", appService.ConfigFolder())
}
