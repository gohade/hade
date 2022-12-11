package command

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v39/github"
	"net/http"
	"net/url"
	"strings"

	"github.com/gohade/hade/framework/cobra"
	"github.com/gohade/hade/framework/contract"
)

// initEnvCommand 获取env相关的命令
func initVersionCommand() *cobra.Command {
	versionCommand.AddCommand(versionListCommand)
	return versionCommand
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "当前hade的版本",
	Run: func(c *cobra.Command, args []string) {
		container := c.GetContainer()
		appService := container.MustMake(contract.AppKey).(contract.App)
		fmt.Println("hade version:", appService.Version())
	},
}

var versionListCommand = &cobra.Command{
	Use:   "list",
	Short: "获取最新的hade的版本",
	RunE: func(c *cobra.Command, args []string) error {
		// 检测到github的连接
		fmt.Println("===============前置条件检测===============")
		fmt.Println("hade源码从github.com中下载，正在检测到github.com的连接")
		var client *github.Client
		client = github.NewClient(nil)
		perPage := 10
		opts := &github.ListOptions{Page: 1, PerPage: perPage}
		releases, rsp, err := client.Repositories.ListReleases(context.Background(), "gohade", "hade", opts)
		fmt.Println(rsp.Rate.String())
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				fmt.Println("错误提示：" + err.Error())
				fmt.Println("说明你的出口ip遇到github的调用限制，可以使用github.com帐号登录方式来增加调用次数")
				githubUserName := ""
				prompt := &survey.Input{
					Message: "请输入github帐号用户名：",
				}
				if err := survey.AskOne(prompt, &githubUserName); err != nil {
					fmt.Println("任务终止：" + err.Error())
					return nil
				}
				githubPassword := ""
				promptPwd := &survey.Password{
					Message: "请输入github帐号密码：",
				}
				if err := survey.AskOne(promptPwd, &githubPassword); err != nil {
					fmt.Println("任务终止：" + err.Error())
					return nil
				}

				httpClient := &http.Client{
					Transport: &http.Transport{
						Proxy: func(req *http.Request) (*url.URL, error) {
							req.SetBasicAuth(githubUserName, githubPassword)
							return nil, nil
						},
					},
				}
				client = github.NewClient(httpClient)
				releases, rsp, err = client.Repositories.ListReleases(context.Background(), "gohade", "hade", opts)
				if err != nil {
					fmt.Println("错误提示：" + err.Error())
					fmt.Println("用户名密码错误，请重新开始")
					return nil
				}
				if len(releases) == 0 {
					fmt.Println("用户名密码错误，请重新开始")
					return nil
				}
				fmt.Println(rsp.Rate.String())
			} else {
				fmt.Println("github.com的连接异常：" + err.Error())
				return nil
			}
		}
		fmt.Println("hade源码从github.com中下载，github.com的连接正常")
		// 这里下面的client都是可用的了
		if rsp.LastPage != 0 {
			opts = &github.ListOptions{Page: rsp.LastPage, PerPage: perPage}
			releases, rsp, err = client.Repositories.ListReleases(context.Background(), "gohade", "hade", opts)
			if err != nil {
				fmt.Println("任务终止：" + err.Error())
				return nil
			}
			fmt.Println(rsp.Rate.String())
		}
		fmt.Println("===============前置条件检测结束===============")
		fmt.Printf("\n")
		fmt.Printf("最新的%v个版本\n", len(releases))
		for _, releaseTmp := range releases {
			fmt.Println("-" + releaseTmp.GetTagName())
			fmt.Println("  发布时间：" + releaseTmp.GetPublishedAt().Format("2006-01-02 15:04:05"))
			fmt.Println("  修改说明：")
			fmt.Println("    " + strings.ReplaceAll(releaseTmp.GetBody(), "\n", "\n    "))
		}
		fmt.Printf("\n")

		fmt.Printf("更多历史版本请参考 https://github.com/gohade/hade/releases\n")
		return nil
	},
}
