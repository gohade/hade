package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/erikdubbelboer/gspt"
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

func startAgentServe(server *http.Server, closeWait time.Duration) error {
	return serveAgent(server, server.ListenAndServe, closeWait)
}

func serveAgent(server *http.Server, serve func() error, closeWait time.Duration) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- serve()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(quit)

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-quit:
		timeoutCtx, cancel := context.WithTimeout(context.Background(), closeWait)
		defer cancel()
		if err := server.Shutdown(timeoutCtx); err != nil {
			return err
		}
		err := <-serveErr
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newAgentStartCommand(options *agentOptions) *cobra.Command {
	return &cobra.Command{
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
			address := resolveAgentAddress(options.address, envService.Get("AGENT_ADDRESS"), configPort, configExists)
			closeWait := 5 * time.Second
			if configService.IsExist("agent.close_wait") {
				closeWait = time.Duration(configService.GetInt("agent.close_wait")) * time.Second
			}

			appService := container.MustMake(contract.AppKey).(contract.App)
			pidFolder := appService.RuntimeFolder()
			if !util.Exists(pidFolder) {
				if err := os.MkdirAll(pidFolder, os.ModePerm); err != nil {
					return err
				}
			}
			logFolder := appService.LogFolder()
			if !util.Exists(logFolder) {
				if err := os.MkdirAll(logFolder, os.ModePerm); err != nil {
					return err
				}
			}

			serverPidFile := filepath.Join(pidFolder, "agent.pid")
			serverLogFile := filepath.Join(logFolder, "agent.log")
			processName := "hade agent"
			executable := "hade"
			if len(os.Args) > 0 {
				executable = os.Args[0]
				processName = filepath.Base(executable) + " agent"
			}
			server := &http.Server{Handler: core, Addr: address}

			if options.daemon {
				probe, err := net.Listen("tcp", address)
				if err != nil {
					return err
				}
				if err := probe.Close(); err != nil {
					return err
				}

				cntxt := &daemon.Context{
					PidFileName: serverPidFile,
					PidFilePerm: 0664,
					LogFileName: serverLogFile,
					LogFilePerm: 0640,
					WorkDir:     util.GetExecDirectory(),
					Umask:       027,
					Args:        buildDaemonArgs(executable, options),
					Env:         os.Environ(),
				}
				d, err := cntxt.Reborn()
				if err != nil {
					return err
				}
				if d != nil {
					printAgentStarted(processName, strconv.Itoa(d.Pid), address, appService)
					return nil
				}
				defer cntxt.Release()
				currentPID := os.Getpid()
				defer cleanupPIDFile(serverPidFile, currentPID)
				gspt.SetProcTitle(processName)
				return startAgentServe(server, closeWait)
			}

			listener, err := net.Listen("tcp", address)
			if err != nil {
				return err
			}
			defer listener.Close()

			currentPID := os.Getpid()
			content := strconv.Itoa(currentPID)
			if err := os.WriteFile(serverPidFile, []byte(content), 0644); err != nil {
				return err
			}
			defer cleanupPIDFile(serverPidFile, currentPID)

			gspt.SetProcTitle(processName)
			printAgentStarted(processName, content, address, appService)
			return serveAgent(server, func() error { return server.Serve(listener) }, closeWait)
		},
	}
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
