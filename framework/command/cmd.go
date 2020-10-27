package command

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"
	"hade/framework/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jianfengye/collection"
	"github.com/pkg/errors"
)

var cmdCommand = &cobra.Command{
	Use:   "command",
	Short: "all about commond",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

var cmdListCommand = &cobra.Command{
	Use:   "list",
	Short: "show all command list",
	RunE: func(c *cobra.Command, args []string) error {
		cmds := c.Root().Commands()
		ps := [][]string{}
		for _, cmd := range cmds {
			line := []string{cmd.Name(), cmd.Short}
			ps = append(ps, line)
		}
		util.PrettyPrint(ps)
		return nil
	},
}

var cmdCreateCommand = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create", "init"},
	Short:   "create a command",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())

		fmt.Println("create a new command...")
		var name string
		var file string
		{
			prompt := &survey.Input{
				Message: "please input command name:",
			}
			err := survey.AskOne(prompt, &name)
			if err != nil {
				return err
			}
		}
		{
			prompt := &survey.Input{
				Message: "please input file name(default: command name):",
			}
			err := survey.AskOne(prompt, &file)
			if err != nil {
				return err
			}
		}

		if file == "" {
			file = name
		}

		// 判断文件不存在
		app := container.MustMake(contract.AppKey).(contract.App)

		pFolder := filepath.Join(app.BasePath(), "app", "console", "command")
		subFolders, err := util.SubDir(pFolder)
		if err != nil {
			return err
		}
		subColl := collection.NewStrCollection(subFolders)
		if subColl.Contains(file + ".go") {
			fmt.Println("the file is existed")
			return nil
		}

		// 开始创建文件
		funcs := template.FuncMap{"title": strings.Title}
		cmdFile := filepath.Join(pFolder, file+".go")
		f, err := os.Create(cmdFile)
		if err != nil {
			return errors.Cause(err)
		}

		t := template.Must(template.New("cmd").Funcs(funcs).Parse(cmdTmpl))
		if err := t.Execute(f, name); err != nil {
			return errors.Cause(err)
		}

		fmt.Println("create new command success，file path:", filepath.Join(pFolder, file+".go"))
		fmt.Println("please remember add command to console/kernel.go")
		return nil
	},
}

var cmdTmpl string = `package command

import (
	"fmt"

	"hade/framework/cobra"
	"hade/framework/command/util"
)

var {{.|title}}Command = &cobra.Command{
	Use:   "{{.}}",
	Short: "{{.}}",
	RunE: func(c *cobra.Command, args []string) error {
		container := util.GetContainer(c.Root())
		fmt.Println(container)
		return nil
	},
}

`
