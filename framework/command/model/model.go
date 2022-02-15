package model

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/orm"
	"github.com/gohade/hade/framework/util"
	"github.com/jianfengye/collection"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"io/ioutil"
	"path/filepath"
)

// 代表输出路径
var output string

// 代表数据库连接
var database string

// InitModelCommand 获取model相关的命令
func InitModelCommand() *cobra.Command {
	modelGenCommand.Flags().StringVarP(&output, "output", "o", "", "模型输出地址")
	modelGenCommand.MarkFlagRequired("output")
	modelGenCommand.Flags().StringVarP(&database, "database", "d", "database.default", "模型连接的数据库")

	modelCommand.AddCommand(modelGenCommand)
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

// modelGenCommand 生成数据库模型的model代码文件
var modelGenCommand = &cobra.Command{
	Use:   "gen",
	Short: "生成模型",
	RunE: func(c *cobra.Command, args []string) error {

		// 确认output路径是绝对路径
		if !filepath.IsAbs(output) {
			absOutput, err := filepath.Abs(output)
			if err != nil {
				return err
			}
			output = absOutput
		}

		// 获取env环境
		container := c.GetContainer()
		logger := container.MustMake(contract.LogKey).(contract.Log)
		logger.SetLevel(contract.ErrorLevel)

		gormService := container.MustMake(contract.ORMKey).(contract.ORMService)
		db, err := gormService.GetDB(orm.WithConfigPath(database))
		if err != nil {
			fmt.Println("数据库连接：" + database + "失败，请检查配置")
			return err
		}

		// 获取所有表
		dbTables, err := db.Migrator().GetTables()
		if err != nil {
			return err
		}

		tables := make([]string, 0, len(dbTables)+1)
		tables = append(tables, "*")
		tables = append(tables, dbTables...)

		genTables := make([]string, 0, len(dbTables)+1)
		{
			// 第一步是一个交互命令行工具，首先展示要生成的表列表选择：
			prompt := &survey.MultiSelect{
				Message: "请选择要生成模型的表格：",
				Options: tables,
			}
			survey.AskOne(prompt, &genTables)

			if collection.NewStrCollection(genTables).Contains("*") {
				genTables = dbTables
			}
		}
		if len(genTables) == 0 {
			return errors.New("未选择任何需要生成模型的数据表")
		}

		// 第二步确认要生成的目录和文件，以及覆盖提示：
		existFileNames := []string{}
		if util.Exists(output) {
			files, err := ioutil.ReadDir(output)
			if err != nil {
				return err
			}
			for _, file := range files {
				existFileNames = append(existFileNames, file.Name())
			}
		}

		genFileNames := make([]string, 0, len(genTables))
		for _, genTable := range genTables {
			genFileNames = append(genFileNames, genTable+".gen.go")
		}
		existFileNamesColl := collection.NewStrCollection(existFileNames)
		genFileNamesColl := collection.NewStrCollection(genFileNames)

		createFileNamesColl := genFileNamesColl.Diff(existFileNamesColl)
		replaceFileNamesColl := genFileNamesColl.Intersect(existFileNamesColl)

		fmt.Println("继续下列操作会在目录（" + output + "）生成下列文件：")
		for _, genFileName := range genFileNames {
			if createFileNamesColl.Contains(genFileName) {
				fmt.Println(genFileName + "（新文件）")
			} else if replaceFileNamesColl.Contains(genFileName) {
				fmt.Println(genFileName + "（覆盖）")
			} else {
				fmt.Println(genFileName)
			}
		}

		genContinue := false
		prompt := &survey.Confirm{
			Message: "请确认是否继续？",
		}
		survey.AskOne(prompt, &genContinue)

		if genContinue == false {
			fmt.Println("操作暂停")
			return nil
		}

		// 第三步选择后是一个生成模型的选项：
		selectRuleTips := []string{}
		ruleTips := map[string]string{
			"FieldNullable":     "FieldNullable, 对于数据库的可null字段设置指针",
			"FieldCoverable":    "FieldCoverable, 根据数据库的Default设置字段的默认值",
			"FieldWithIndexTag": "FieldWithIndexTag, 根据数据库的索引关系设置索引标签",
			"FieldWithTypeTag":  "FieldWithTypeTag, 生成类型字段",
		}
		tips := make([]string, 0, len(ruleTips))
		for _, val := range ruleTips {
			tips = append(tips, val)
		}
		promptRules := &survey.MultiSelect{
			Message: "请选择生成的模型规则：",
			Options: tips,
		}
		survey.AskOne(promptRules, &selectRuleTips)
		isSelectRule := func(key string, selectRuleTips []string, allRuleTips map[string]string) bool {
			tip := allRuleTips[key]
			selectRuleTipsColl := collection.NewStrCollection(selectRuleTips)
			return selectRuleTipsColl.Contains(tip)
		}

		// 生成模型文件

		g := gen.NewGenerator(gen.Config{
			ModelPkgPath: output,

			FieldNullable:     isSelectRule("FieldNullable", selectRuleTips, ruleTips),
			FieldCoverable:    isSelectRule("FieldCoverable", selectRuleTips, ruleTips),
			FieldWithIndexTag: isSelectRule("FieldWithIndexTag", selectRuleTips, ruleTips),
			FieldWithTypeTag:  isSelectRule("FieldWithTypeTag", selectRuleTips, ruleTips),

			Mode: gen.WithDefaultQuery,
		})

		g.UseDB(db)

		for _, table := range genTables {
			g.GenerateModel(table)
		}
		g.Execute()

		fmt.Println("生成模型成功")
		return nil
	},
}
