package command

import (
	"fmt"

	"hade/framework/cobra"
	"hade/framework/command/util"
	"hade/framework/contract"
)

// envCommand show current envionment
var envCommand = &cobra.Command{
	Use:   "env",
	Short: "get current environment",
	Run: func(c *cobra.Command, args []string) {
		container := util.GetContainer(c.Root())
		envService := container.MustMake(contract.EnvKey).(contract.Env)
		fmt.Println("environment:", envService.AppEnv())
	},
}
