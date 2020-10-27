package command

import (
	"log"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/command/util"
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
