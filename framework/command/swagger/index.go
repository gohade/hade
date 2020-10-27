package swagger

import "hade/framework/cobra"

var IndexCommand = &cobra.Command{
	Use:   "swagger",
	Short: "swagger operator",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}
