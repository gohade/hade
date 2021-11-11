package command

import (
	"fmt"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/kr/pretty"
)

// initConfigCommand 获取配置相关的命令
func initConfigCommand() *cobra.Command {
	configCommand.AddCommand(configGetCommand)
	return configCommand
}

// envCommand 获取当前的App环境
var configCommand = &cobra.Command{
	Use:   "config",
	Short: "获取配置相关信息",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

// envListCommand 获取所有的App环境变量
var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "获取某个配置信息",
	RunE: func(c *cobra.Command, args []string) error {
		container := c.GetContainer()
		configService := container.MustMake(contract.ConfigKey).(contract.Config)
		if len(args) != 1 {
			fmt.Println("参数错误")
			return nil
		}
		configPath := args[0]
		val := configService.Get(configPath)
		if val == nil {
			fmt.Println("配置路径 ", configPath, " 不存在")
			return nil
		}

		fmt.Printf("%# v\n", pretty.Formatter(val))
		return nil
	},
}
