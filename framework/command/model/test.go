package model

import (
	"fmt"
	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/orm"
)

// modelTestCommand 检测数据库连接是否正常
var modelTestCommand = &cobra.Command{
	Use:          "test",
	Short:        "测试数据库",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
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
			fmt.Println("数据库连接：" + database + "失败，请检查配置")
			return err
		}
		fmt.Println("数据库连接：" + database + "成功")

		// 一共存在多少张表
		fmt.Printf("一共存在%d张表\n", len(dbTables))
		for _, table := range dbTables {
			fmt.Println(table)
		}
		return nil
	},
}
