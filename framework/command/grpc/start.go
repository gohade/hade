package grpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/erikdubbelboer/gspt"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/util"
	"github.com/gohade/hade/framework/util/goroutine"
	"github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
)

// grpcStartCommand 启动一个Web服务
var grpcStartCommand = &cobra.Command{
	Use:   "start",
	Short: "启动一个grpc服务",
	RunE: func(c *cobra.Command, args []string) error {
		// 从Command中获取服务容器
		container := c.GetContainer()
		// 从服务容器中获取kernel的服务实例
		kernelService := container.MustMake(contract.KernelKey).(contract.Kernel)
		ctx := context.Background()

		// 先读取参数，然后读取Env，然后读取配置文件
		if appAddress == "" {
			envService := container.MustMake(contract.EnvKey).(contract.Env)
			if envService.Get("ADDRESS") != "" {
				appAddress = envService.Get("ADDRESS")
			} else {
				configService := container.MustMake(contract.ConfigKey).(contract.Config)
				if configService.IsExist("grpc.address") {
					appAddress = configService.GetString("grpc.address")
				} else {
					appAddress = ":8888"
				}
			}
		}
		server := kernelService.GrpcEngine()

		// 从kernel服务实例中获取引擎
		lis, err := net.Listen("tcp", appAddress)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		appService := container.MustMake(contract.AppKey).(contract.App)
		pidFolder := appService.RuntimeFolder()
		if !util.Exists(pidFolder) {
			if err := os.MkdirAll(pidFolder, os.ModePerm); err != nil {
				return err
			}
		}
		serverPidFile := filepath.Join(pidFolder, "grpc.pid")
		logFolder := appService.LogFolder()
		if !util.Exists(logFolder) {
			if err := os.MkdirAll(logFolder, os.ModePerm); err != nil {
				return err
			}
		}
		processName := "hade grpc"
		if len(os.Args) > 0 {
			processName = filepath.Base(os.Args[0]) + " grpc"
		}
		// 应用日志
		serverLogFile := filepath.Join(logFolder, "grpc.log")
		currentFolder := util.GetExecDirectory()
		// daemon 模式
		if appDaemon {
			parentArgs := make([]string, 0, len(os.Args))
			for _, arg := range os.Args {
				if strings.HasPrefix(arg, "--") {
					if strings.Contains(arg, "--daemon=") {
						continue
					}
					parentArgs = append(parentArgs, arg)
				}
			}
			subArgs := []string{filepath.Base(os.Args[0]), "grpc", "start", "--daemon=true"}
			subArgs = append(subArgs, parentArgs...)
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
				// 子进程的参数，按照这个参数设置，子进程的命令为 ./hade app start --daemon=true
				Args: subArgs,
				// 环境变量和父进程一样
				Env: os.Environ(),
			}
			// 启动子进程，d不为空表示当前是父进程，d为空表示当前是子进程
			d, err := cntxt.Reborn()
			if err != nil {
				return err
			}
			if d != nil {
				// 父进程直接打印启动成功信息，不做任何操作
				fmt.Println("成功启动进程:", processName)
				fmt.Println("进程pid:", d.Pid)
				showAppAddress := appAddress
				if strings.HasPrefix(appAddress, ":") {
					showAppAddress = "grpc://localhost" + showAppAddress
				}
				fmt.Println("监听地址:", showAppAddress)
				fmt.Println("基础路径:", appService.BaseFolder())
				fmt.Println("日志路径:", appService.LogFolder())
				fmt.Println("运行路径:", appService.RuntimeFolder())
				fmt.Println("配置路径:", appService.ConfigFolder())
				return nil
			}
			defer cntxt.Release()
			// 子进程执行真正的app启动操作
			fmt.Println("daemon started")
			gspt.SetProcTitle(processName)
			if err := startAppServe(ctx, server, lis, container); err != nil {
				fmt.Println(err)
			}
			return nil
		}

		// 非deamon模式，直接执行
		content := strconv.Itoa(os.Getpid())
		fmt.Println("成功启动进程:", processName)
		fmt.Println("进程pid:", content)
		err = ioutil.WriteFile(serverPidFile, []byte(content), 0644)
		if err != nil {
			return err
		}
		gspt.SetProcTitle(processName)
		showAppAddress := appAddress
		if strings.HasPrefix(appAddress, ":") {
			showAppAddress = "grpc://localhost" + showAppAddress
		}
		fmt.Println("监听地址:", showAppAddress)
		fmt.Println("基础路径:", appService.BaseFolder())
		fmt.Println("日志路径:", appService.LogFolder())
		fmt.Println("运行路径:", appService.RuntimeFolder())
		fmt.Println("配置路径:", appService.ConfigFolder())

		if err := startAppServe(ctx, server, lis, container); err != nil {
			fmt.Println(err)
		}
		return nil
	},
}

// 启动AppServer, 这个函数会将当前goroutine阻塞
func startAppServe(ctx context.Context, server *grpc.Server, lis net.Listener, c framework.Container) error {

	logger := c.MustMake(contract.LogKey).(contract.Log)
	// 这个goroutine是启动服务的goroutine
	goroutine.SafeGo(ctx, func() {
		if err := server.Serve(lis); err != nil {
			logger.Error(ctx, "grpc serve error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	})

	// 当前的goroutine等待信号量
	quit := make(chan os.Signal)
	// 监控信号：SIGINT, SIGTERM, SIGQUIT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 这里会阻塞当前goroutine等待信号
	<-quit

	configService := c.MustMake(contract.ConfigKey).(contract.Config)

	// 默认是grace stop
	forceStop := false
	// 调用Server.Shutdown graceful结束
	if configService.IsExist("grpc.force_stop") {
		forceStop = configService.GetBool("grpc.force_stop")
	}

	if forceStop {
		server.Stop()
	} else {
		server.GracefulStop()
	}
	return nil
}
