package grpc

import "github.com/gohade/hade/framework/cobra"

// app启动地址
var appAddress = ""
var appDaemon = false

// appBaseArgs对应AppService中可以配置的参数
var appStartArgs = []string{
	"base_folder",
	"config_folder",
	"log_folder",
	"http_folder",
	"console_folder",
	"storage_folder",
	"provider_folder",
	"middleware_folder",
	"command_folder",
	"runtime_folder",
	"test_folder",
	"deploy_folder",
	"app_folder",
}

var appOtherArgs = []string{
	"runtime_folder",
	"storage_folder",
	"base_folder",
}

// InitGrpcCommand 获取grpc相关的命令
func InitGrpcCommand() *cobra.Command {
	grpcStartCommand.Flags().BoolVarP(&appDaemon, "daemon", "d", false, "开启后台模式")
	grpcStartCommand.Flags().StringVar(&appAddress, "address", "", "设置app启动的地址，默认为:8888")

	for _, arg := range appStartArgs {
		tmp := ""
		grpcStartCommand.Flags().StringVar(&tmp, arg, "", "base config for app service: "+arg)
	}

	for _, arg := range appOtherArgs {
		tmp := ""
		grpcRestartCommand.Flags().StringVar(&tmp, arg, "", "base config for app service: "+arg)
		grpcStateCommand.Flags().StringVar(&tmp, arg, "", "base config for app service: "+arg)
		grpcStopCommand.Flags().StringVar(&tmp, arg, "", "base config for app service: "+arg)
	}

	grpcCommand.AddCommand(grpcStartCommand)
	grpcCommand.AddCommand(grpcRestartCommand)
	grpcCommand.AddCommand(grpcStateCommand)
	grpcCommand.AddCommand(grpcStopCommand)
	return grpcCommand
}

// grpcCommand 模型相关的命令
var grpcCommand = &cobra.Command{
	Use:   "grpc",
	Short: "grpc相关的命令",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}
		return nil
	},
}
