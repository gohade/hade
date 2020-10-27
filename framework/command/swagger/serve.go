package swagger

import (
	"fmt"
	"net/url"
	"path/filepath"

	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"
	"hade/framework/gin"
	ginSwagger "hade/framework/middleware/gin-swagger"
	"hade/framework/middleware/gin-swagger/swaggerFiles"
)

var (
	address string = "http://localhost:8088"
)

// 初始化Dev命令
func InitServeCommand() *cobra.Command {
	ServeCommand.Flags().StringVarP(&address, "address", "a", "http://localhost:8088", "监听地址")
	return ServeCommand
}

// envCommand show current envionment
var ServeCommand = &cobra.Command{
	Use:   "serve",
	Short: "use gen serve",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		r := gin.Default()

		hadeURL, err := url.Parse(address)
		if err != nil {
			return err
		}
		address := fmt.Sprintf("%s:%s", hadeURL.Hostname(), hadeURL.Port())

		jsonFolder := filepath.Join(appService.BasePath(), "app", "http", "swagger")

		r.Static("/swagger_gen/", jsonFolder)
		swaggerGenUrl := ginSwagger.URL("/swagger_gen/swagger.json")
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerGenUrl))

		fmt.Println("swagger success: http://" + address + "/swagger/index.html")
		fmt.Println("if you want to replace, remember use command: swagger gen")
		r.Run(address)
		return nil
	},
}
