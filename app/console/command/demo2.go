package command

import (
	"fmt"

	"github.com/gohade/hade/framework/cobra"
)

var Demo2Command = &cobra.Command{
	Use:   "demo2",
	Short: "demo2",
	RunE: func(c *cobra.Command, args []string) error {
        container := c.GetContainer()
		fmt.Println(container)
		return nil
	},
}

