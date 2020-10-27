package command

import (
	"hade/framework/cobra"
)

// helpCommand show current envionment
var helpCommand = &cobra.Command{
	Use:   "help",
	Short: "get help info",
	Run: func(c *cobra.Command, args []string) {
		cmd, _, e := c.Root().Find(args)
		if cmd == nil || e != nil {
			c.Printf("Unknown help topic %#q\n", args)
			c.Root().Usage()
		} else {
			cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
			cmd.Help()
		}
	},
}
