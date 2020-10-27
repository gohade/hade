package console

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/cobra"
	hadeCommand "github.com/gohade/hade/framework/command"
	commandUtil "github.com/gohade/hade/framework/command/util"
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
