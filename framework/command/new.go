package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"hade/framework/cobra"
	"hade/framework/util"

	"github.com/pkg/errors"
)

var newForce bool
var newGoMod string

func initNewCommand() *cobra.Command {
	newCommand.Flags().BoolVarP(&newForce, "force", "f", false, "if app exist, overwrite app, default: false")
	newCommand.Flags().StringVarP(&newGoMod, "mod", "m", "", "go mod name, default: folder name")
	return newCommand
}

var newCommand = &cobra.Command{
	Use:     "new [folder]",
	Aliases: []string{"create", "init"},
	Short:   "create a new app",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("参数错误")
		}
		name := args[0]

		currentPath := util.GetExecDirectory()

		folder := filepath.Join(currentPath, name)
		if util.Exists(folder) {
			if newForce {
				os.RemoveAll(folder)
			} else {
				return errors.New("app has exist, please delete first")
			}
		}

		if newGoMod == "" {
			newGoMod = name
		}

		// 拷贝template项目
		url := "https://hade-template/-/archive/master/hade-template-master.zip"
		err := util.DownloadFile("hade-template-master.zip", url)
		if err != nil {
			return err
		}

		_, err = util.Unzip("hade-template-master.zip", "/tmp/")
		if err != nil {
			return err
		}

		// TODO: check do not use tmp file
		if err := os.Rename("/tmp/hade-template-master/", folder); err != nil {
			return err
		}

		if err := os.Remove("hade-template-master.zip"); err != nil {
			return err
		}
		fmt.Println("remove " + path.Join(folder, ".git"))
		os.RemoveAll(path.Join(folder, ".git"))

		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			fmt.Println("read file:" + path)
			if info.IsDir() {
				return nil
			}

			c, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			isContain := bytes.Contains(c, []byte("{{hade_project_name}}"))
			if isContain {
				fmt.Println("update file:" + path)
				c = bytes.ReplaceAll(c, []byte("{{hade_project_name}}"), []byte(newGoMod))
				err = ioutil.WriteFile(path, c, 0644)
				if err != nil {
					return err
				}
			}

			return nil
		})
		return nil
	},
}
