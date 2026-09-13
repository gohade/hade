package agent

import (
	"path/filepath"

	"github.com/gohade/hade/framework/cobra"
)

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

type agentOptions struct {
	address      string
	daemon       bool
	folderValues map[string]*string
}

func newAgentOptions() *agentOptions {
	options := &agentOptions{folderValues: make(map[string]*string, len(agentStartArgs))}
	for _, name := range agentStartArgs {
		value := ""
		options.folderValues[name] = &value
	}
	return options
}

type agentDependencies struct {
	process processOperations
}

// InitAgentCommand 每次创建独立的 Agent 命令树与选项状态。
func InitAgentCommand() *cobra.Command {
	options := newAgentOptions()
	deps := agentDependencies{
		process: defaultProcessOperations(),
	}
	startCommand := newAgentStartCommand(options, deps)
	stopCommand := newAgentStopCommand(deps)
	restartCommand := newAgentRestartCommand(options, deps)
	stateCommand := newAgentStateCommand()
	agentCommand := newAgentRootCommand()

	startCommand.Flags().BoolVarP(&options.daemon, "daemon", "d", false, "开启后台模式")
	startCommand.Flags().StringVar(&options.address, "address", "", "设置 agent 启动地址，默认为 :8889")
	for _, arg := range agentStartArgs {
		startCommand.Flags().StringVar(options.folderValues[arg], arg, "", "base config for agent service: "+arg)
	}
	for _, arg := range agentOtherArgs {
		restartCommand.Flags().StringVar(options.folderValues[arg], arg, "", "base config for agent service: "+arg)
		stateCommand.Flags().String(arg, "", "base config for agent service: "+arg)
		stopCommand.Flags().String(arg, "", "base config for agent service: "+arg)
	}

	agentCommand.AddCommand(startCommand)
	agentCommand.AddCommand(restartCommand)
	agentCommand.AddCommand(stateCommand)
	agentCommand.AddCommand(stopCommand)
	return agentCommand
}

func newAgentRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "agent 相关的命令",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return nil
		},
	}
}

func buildDaemonArgs(executable string, options *agentOptions) []string {
	args := []string{filepath.Base(executable), "agent", "start", "--daemon=true"}
	if options.address != "" {
		args = append(args, "--address", options.address)
	}
	for _, name := range agentStartArgs {
		if value := *options.folderValues[name]; value != "" {
			args = append(args, "--"+name, value)
		}
	}
	return args
}
