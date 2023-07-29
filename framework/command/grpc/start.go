package grpc

import (
	"log"
	"net"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
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

		// 从kernel服务实例中获取引擎
		lis, err := net.Listen("tcp", appAddress)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		server := kernelService.GrpcEngine()
		if err := server.Serve(lis); err != nil {
			return err
		}
		return nil

	},
}
