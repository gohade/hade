package grpc

import "github.com/gohade/hade/framework/cobra"

// app启动地址
var appAddress = ""
var appDaemon = false

// InitGrpcCommand 获取grpc相关的命令
func InitGrpcCommand() *cobra.Command {
	grpcStartCommand.Flags().BoolVarP(&appDaemon, "daemon", "d", false, "开启后台模式")
	grpcStartCommand.Flags().StringVar(&appAddress, "address", "", "设置app启动的地址，默认为:8888")

	grpcCommand.AddCommand(grpcStartCommand)
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
