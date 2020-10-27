package console

import (
	"hade/framework"
	"hade/framework/cobra"
	hadeCommand "hade/framework/command"
	commandUtil "hade/framework/command/util"
)

// RunCommand is command
func RunCommand(container framework.Container) error {
	var rootCmd = &cobra.Command{
		Use:   "hade",
		Short: "main",
		Long:  "hade commands",
	}

	ctx := commandUtil.RegiestContainer(container, rootCmd)

	hadeCommand.AddKernelCommands(rootCmd)

	// rootCmd.AddCronCommand("* * * * *", command.DemoCommand)

	return rootCmd.ExecuteContext(ctx)
}
