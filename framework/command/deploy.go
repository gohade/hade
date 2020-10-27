package command

import (
	"fmt"
	"path/filepath"

	"hade/framework/cobra"
	"hade/framework/command/util"
	"hade/framework/contract"

	"github.com/pkg/errors"
)

// envCommand show current envionment
var deployCommand = &cobra.Command{
	Use:   "deploy",
	Short: "deploy app by ssh",
	RunE: func(c *cobra.Command, args []string) error {
		container := util.GetContainer(c.Root())
		app := container.MustMake(contract.AppKey).(contract.App)
		config := container.MustMake(contract.ConfigKey).(contract.Config)

		sshConfig := &contract.SSHConfig{
			User:     config.GetString("deploy.user"),
			Password: config.GetString("deploy.password"),
			Host:     config.GetString("deploy.ssh"),
			Port:     config.GetString("deploy.port"),
			RsaKey:   config.GetString("deploy.rsa_key"),
			Timeout:  config.GetInt("deploy.timeout"),
		}
		ish, err := container.MakeNew(contract.SSHKey, []interface{}{sshConfig})
		if err != nil {
			return err
		}
		sh := ish.(contract.SSH)

		if err := buildAllCommand.RunE(c, nil); err != nil {
			return err
		}

		remotePath := config.GetString("deploy.remote_path")
		if remotePath == "" {
			return errors.New("remote_path error")
		}

		// 配置文件更新
		srcConfigFolder := app.ConfigPath()
		distConfigFolder := remotePath
		if err := sh.UploadDir(srcConfigFolder, distConfigFolder); err != nil {
			return err
		}

		// hade执行文件更新
		srcHadeFile := filepath.Join(app.BasePath(), "hade")
		distHadeFile := filepath.Join(remotePath, "hade")
		if err := sh.Upload(srcHadeFile, distHadeFile); err != nil {
			return err
		}

		// 前端文件更新
		srcFrontFolder := filepath.Join(app.BasePath(), "dist")
		distFrontFolder := filepath.Join(remotePath, "dist")
		if err := sh.UploadDir(srcFrontFolder, distFrontFolder); err != nil {
			return err
		}

		// 执行post
		if config.IsExist("deploy.post_shell") {
			shells := config.GetStringSlice("deploy.post_shell")
			for _, shell := range shells {
				out, err := sh.Run(shell)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
			}
		}
		return nil
	},
}
