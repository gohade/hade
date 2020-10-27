package command

import (
	"log"

	"hade/framework/cobra"
	"hade/framework/command/util"
)

var DemoCommand = &cobra.Command{
	Use:   "demo",
	Short: "demo",
	RunE: func(c *cobra.Command, args []string) error {
		container := util.GetContainer(c.Root())
		log.Println(container)
		return nil
	},
}
