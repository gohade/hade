package foo

import (
	"fmt"

	"github.com/gohade/hade/framework/cobra"
)

var FooCommand = &cobra.Command{
	Use:   "foo",
	Short: "foo",
	RunE: func(c *cobra.Command, args []string) error {
		fmt.Println("this is foo command")
		return nil
	},
}
