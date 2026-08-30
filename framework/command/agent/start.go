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
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
	"github.com/sevlyar/go-daemon"
)

const (
	agentDaemonAuthFDEnv = "HADE_AGENT_DAEMON_AUTH_FD"
	agentDaemonAuthFD    = 4
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

type listenFunc func(network, address string) (net.Listener, error)

func listenAndMarkReady(readyFile string, pid int, address string, listen listenFunc) (net.Listener, error) {
	listener, err := listen("tcp", address)
	if err != nil {
		return nil, err
	}
	if err := writeReadyFile(readyFile, pid); err != nil {
		return nil, mergeErrors(err, closeNetworkListener(listener))
	}
	return listener, nil
}

func closeNetworkListener(listener net.Listener) error {
	err := listener.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func daemonChildEnvironment(base []string, fd int) []string {
	env := append([]string{}, base...)
	return append(env, agentDaemonAuthFDEnv+"="+strconv.Itoa(fd))
}

type agentRuntime struct {
	container      framework.Container
	appService     contract.App
	server         *http.Server
	address        string
	closeWait      time.Duration
	pidFile        string
	readyFile      string
	lifecycleFile  string
	daemonAuthFile string
	logFile        string
	processName    string
	executable     string
}

func prepareAgentRuntime(c *cobra.Command, options *agentOptions) (*agentRuntime, error) {
	container := c.GetContainer()
	kernelService := container.MustMake(contract.KernelKey).(contract.Kernel)
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
	if err := os.MkdirAll(appService.RuntimeFolder(), os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(appService.LogFolder(), os.ModePerm); err != nil {
		return nil, err
	}
	executable := "hade"
	if len(os.Args) > 0 {
		executable = os.Args[0]
	}
	return &agentRuntime{
		container:      container,
		appService:     appService,
		server:         &http.Server{Handler: kernelService.AgentEngine(), Addr: address},
		address:        address,
		closeWait:      closeWait,
		pidFile:        filepath.Join(appService.RuntimeFolder(), "agent.pid"),
		readyFile:      filepath.Join(appService.RuntimeFolder(), "agent.ready"),
		lifecycleFile:  filepath.Join(appService.RuntimeFolder(), "agent.lifecycle.lock"),
		daemonAuthFile: filepath.Join(appService.RuntimeFolder(), "agent.daemon.auth"),
		logFile:        filepath.Join(appService.LogFolder(), "agent.log"),
		processName:    filepath.Base(executable) + " agent",
		executable:     executable,
	}, nil
}

func newAgentStartCommand(options *agentOptions, deps agentDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "启动一个 agent 服务",
		RunE: func(c *cobra.Command, args []string) error {
			runtime, err := prepareAgentRuntime(c, options)
			if err != nil {
				return err
			}
			authorization, err := resolveDaemonAuthorization(
				runtime.daemonAuthFile,
				os.LookupEnv,
				deps.process.wasReborn,
			)
			if err != nil {
				return err
			}
			if authorization != nil {
				runErr := startAgentLocked(runtime, options, nil, authorization)
				return mergeErrors(runErr, authorization.discard())
			}
			return withLifecycleLock(runtime.lifecycleFile, func(lock *exclusiveFileLock) error {
				return startAgentLocked(runtime, options, lock, nil)
			})
		},
	}
}

func startAgentLocked(
	runtime *agentRuntime,
	options *agentOptions,
	lifecycle *exclusiveFileLock,
	authorization *daemonAuthorization,
) error {
	releaseAuthorization := func() error { return nil }
	if options.daemon {
		cntxt := &daemon.Context{
			PidFileName: runtime.daemonAuthFile,
			PidFilePerm: 0600,
			LogFileName: runtime.logFile,
			LogFilePerm: 0640,
			WorkDir:     util.GetExecDirectory(),
			Umask:       027,
			Args:        buildDaemonArgs(runtime.executable, options),
			Env:         daemonChildEnvironment(os.Environ(), agentDaemonAuthFD),
		}
		child, err := cntxt.Reborn()
		if err != nil {
			return err
		}
		if child != nil {
			probes := defaultDaemonReadinessProbes(runtime.pidFile, runtime.readyFile, child.Pid)
			if err := waitDaemonOrAbort(
				child.Pid,
				5*time.Second,
				probes,
				daemonChildOperations{
					terminate: func() error { return child.Signal(syscall.SIGTERM) },
					kill:      child.Kill,
					wait: func() error {
						_, err := child.Wait()
						return err
					},
				},
				func() error {
					return cleanupFailedDaemonFiles(
						runtime.pidFile,
						runtime.readyFile,
						runtime.daemonAuthFile,
						child.Pid,
					)
				},
			); err != nil {
				return err
			}
			printAgentStarted(runtime.processName, strconv.Itoa(child.Pid), runtime.address, runtime.appService)
			return nil
		}
		if authorization == nil {
			return errors.New("daemon child 未验证继承 authorization fd")
		}
		releaseAuthorization = func() error {
			return mergeErrors(cntxt.Release(), authorization.discard())
		}
	}

	currentPID := os.Getpid()
	owner, err := acquirePIDFile(runtime.pidFile, currentPID)
	if err != nil {
		return mergeErrors(err, releaseAuthorization())
	}

	listener, err := listenAndMarkReady(runtime.readyFile, currentPID, runtime.address, net.Listen)
	if err != nil {
		return mergeErrors(err, owner.cleanup(), releaseAuthorization())
	}

	if err := lifecycle.release(); err != nil {
		return finishAgentServe(
			mergeErrors(err, releaseAuthorization()),
			func() error { return closeNetworkListener(listener) },
			func() error { return cleanupReadyFile(runtime.readyFile, currentPID) },
			owner.cleanup,
		)
	}
	if err := releaseAuthorization(); err != nil {
		return finishAgentServe(
			err,
			func() error { return closeNetworkListener(listener) },
			func() error { return cleanupReadyFile(runtime.readyFile, currentPID) },
			owner.cleanup,
		)
	}
	gspt.SetProcTitle(runtime.processName)
	if !options.daemon {
		printAgentStarted(runtime.processName, strconv.Itoa(currentPID), runtime.address, runtime.appService)
	}
	serveErr := serveAgent(runtime.server, func() error { return runtime.server.Serve(listener) }, runtime.closeWait)
	return finishAgentServe(
		serveErr,
		func() error { return closeNetworkListener(listener) },
		func() error { return cleanupReadyFile(runtime.readyFile, currentPID) },
		owner.cleanup,
	)
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
