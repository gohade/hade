package model

import (
	"github.com/gohade/hade/framework/cobra"
)

// 代表输出路径
var output string

// 代表数据库连接
var database string

// 代表表格
var table string

// InitModelCommand 获取model相关的命令
func InitModelCommand() *cobra.Command {

	// model gen
	modelGenCommand.Flags().StringVarP(&output, "output", "o", "", "模型输出地址")
	_ = modelGenCommand.MarkFlagRequired("output")
	modelGenCommand.Flags().StringVarP(&database, "database", "d", "database.default", "模型连接的数据库")
	modelCommand.AddCommand(modelGenCommand)

	// model test
	modelTestCommand.Flags().StringVarP(&database, "database", "d", "database.default", "模型连接的数据库")
	modelCommand.AddCommand(modelTestCommand)

	// model api
	modelApiCommand.Flags().StringVarP(&database, "database", "d", "database.default", "模型连接的数据库")
	modelApiCommand.Flags().StringVarP(&table, "table", "t", "default", "模型连接的数据表")
	modelApiCommand.Flags().StringVarP(&output, "output", "o", "", "模型输出地址, 文件夹地址")
	modelCommand.AddCommand(modelApiCommand)
	return modelCommand
}

// modelCommand 模型相关的命令
var modelCommand = &cobra.Command{
	Use:   "model",
	Short: "数据库模型相关的命令",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}
