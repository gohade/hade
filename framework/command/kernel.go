package command

import (
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/command/model"
	"github.com/robfig/cron/v3"
)

// AddKernelCommands will add all command/* to root command
func AddKernelCommands(root *cobra.Command) {
	InitCronCommands(root)

	// app 命令
	root.AddCommand(initAppCommand())
	// env 命令
	root.AddCommand(initEnvCommand())
	// cron 命令
	root.AddCommand(initCronCommand())
	// config 命令
	root.AddCommand(initConfigCommand())
	// build 命令
	root.AddCommand(initBuildCommand())
	// go build
	root.AddCommand(goCommand)
	// npm build
	root.AddCommand(npmCommand)
	// dev
	root.AddCommand(initDevCommand())
	// cmd
	root.AddCommand(initCmdCommand())
	// provider
	root.AddCommand(initProviderCommand())
	// middleware
	root.AddCommand(initMiddlewareCommand())
	// new
	root.AddCommand(initNewCommand())
	// swagger
	root.AddCommand(initSwaggerCommand())
	// deploy
	root.AddCommand(initDeployCommand())
	// model
	root.AddCommand(model.InitModelCommand())
}

// InitCronCommands 初始化Cron相关的命令
func InitCronCommands(root *cobra.Command) {
	// 初始化cron相关命令
	if root.Cron == nil {
		// 初始化cron
		root.Cron = cron.New(cron.WithParser(cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)))
		root.CronSpecs = []cobra.CronSpec{}
	}
}
