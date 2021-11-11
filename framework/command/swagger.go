package command

import (
	"fmt"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/swaggo/swag/gen"
	"path/filepath"
)

func initSwaggerCommand() *cobra.Command {
	swaggerCommand.AddCommand(swaggerGenCommand)
	return swaggerCommand
}

var swaggerCommand = &cobra.Command{
	Use:   "swagger",
	Short: "swagger对应命令",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

// swaggerGenCommand 生成具体的swagger文档
var swaggerGenCommand = &cobra.Command{
	Use:   "gen",
	Short: "生成对应的swagger文件, contain swagger.yaml, doc.go",
	Run: func(c *cobra.Command, args []string) {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)

		outputDir := filepath.Join(appService.AppFolder(), "http", "swagger")
		httpFolder := filepath.Join(appService.AppFolder(), "http")

		conf := &gen.Config{
			// 遍历需要查询注释的目录
			SearchDir: httpFolder,
			// 不包含哪些文件
			Excludes: "",
			// 输出目录
			OutputDir: outputDir,
			// 整个swagger接口的说明文档注释
			MainAPIFile: "swagger.go",
			// 名字的显示策略，比如首字母大写等
			PropNamingStrategy: "",
			// 是否要解析vendor目录
			ParseVendor: false,
			// 是否要解析外部依赖库的包
			ParseDependency: false,
			// 是否要解析标准库的包
			ParseInternal: false,
			// 是否要查找markdown文件，这个markdown文件能用来为tag增加说明格式
			MarkdownFilesDir: "",
			// 是否应该在docs.go中生成时间戳
			GeneratedTime: false,
		}
		err := gen.New().Build(conf)
		if err != nil {
			fmt.Println(err)
		}
	},
}
