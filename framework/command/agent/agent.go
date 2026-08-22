package agent

import "github.com/gohade/hade/framework/cobra"

var agentAddress string
var agentDaemon bool

var agentStartArgs = []string{
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

var agentOtherArgs = []string{
	"runtime_folder",
	"storage_folder",
	"base_folder",
}

// InitAgentCommand 获取 Agent 服务相关命令。
func InitAgentCommand() *cobra.Command {
	agentStartCommand.Flags().BoolVarP(&agentDaemon, "daemon", "d", false, "开启后台模式")
	agentStartCommand.Flags().StringVar(&agentAddress, "address", "", "设置 agent 启动地址，默认为 :8889")

	for _, arg := range agentStartArgs {
		tmp := ""
		agentStartCommand.Flags().StringVar(&tmp, arg, "", "base config for agent service: "+arg)
	}
	for _, arg := range agentOtherArgs {
		tmp := ""
		agentRestartCommand.Flags().StringVar(&tmp, arg, "", "base config for agent service: "+arg)
		agentStateCommand.Flags().StringVar(&tmp, arg, "", "base config for agent service: "+arg)
		agentStopCommand.Flags().StringVar(&tmp, arg, "", "base config for agent service: "+arg)
	}

	agentCommand.AddCommand(agentStartCommand)
	agentCommand.AddCommand(agentRestartCommand)
	agentCommand.AddCommand(agentStateCommand)
	agentCommand.AddCommand(agentStopCommand)
	return agentCommand
}

var agentCommand = &cobra.Command{
	Use:   "agent",
	Short: "agent 相关的命令",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}
		return nil
	},
}
