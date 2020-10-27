package swagger

import (
	"fmt"
	"path/filepath"

	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"

	"github.com/swaggo/swag/gen"
)

// envCommand show current envionment
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "generate swagger file, contain swagger.yaml, doc.go",
	Run: func(c *cobra.Command, args []string) {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		outputDir := filepath.Join(appService.BasePath(), "app", "http", "swagger")

		conf := &gen.Config{
			SearchDir:          "./app/http/",
			Excludes:           "",
			OutputDir:          outputDir,
			MainAPIFile:        "swagger.go",
			PropNamingStrategy: "",
			ParseVendor:        false,
			ParseDependency:    false,
			ParseInternal:      false,
			MarkdownFilesDir:   "",
			GeneratedTime:      false,
		}
		err := gen.New().Build(conf)
		if err != nil {
			fmt.Println(err)
		}
	},
}
