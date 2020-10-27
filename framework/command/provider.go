package command

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"hade/framework"
	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"
	"hade/framework/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jianfengye/collection"
	"github.com/pkg/errors"
)

var providerCommand = &cobra.Command{
	Use:   "provider",
	Short: "about hade service provider",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

var providerListCommand = &cobra.Command{
	Use:   "list",
	Short: "list all installed providers",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		hadeContainer := container.(*framework.HadeContainer)
		list := hadeContainer.PrintList()
		for _, line := range list {
			println(line)
		}
		return nil
	},
}

var providerCreateCommand = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create", "init"},
	Short:   "create a provider",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())

		fmt.Println("create a provider")
		var name string
		var folder string
		{
			prompt := &survey.Input{
				Message: "please input provider name",
			}
			err := survey.AskOne(prompt, &name)
			if err != nil {
				return err
			}
		}
		{
			prompt := &survey.Input{
				Message: "please input provider folder(default: provider name):",
			}
			err := survey.AskOne(prompt, &folder)
			if err != nil {
				return err
			}
		}

		// check is provider exist
		providers := container.(*framework.HadeContainer).PrintList()
		providerColl := collection.NewStrCollection(providers)
		if providerColl.Contains(name) {
			fmt.Println("provider name is existed")
			return nil
		}

		if folder == "" {
			folder = name
		}

		app := container.MustMake(contract.AppKey).(contract.App)

		pFolder := filepath.Join(app.BasePath(), "app", "provider")
		subFolders, err := util.SubDir(pFolder)
		if err != nil {
			return err
		}
		subColl := collection.NewStrCollection(subFolders)
		if subColl.Contains(folder) {
			fmt.Println("provider folder is existed")
			return nil
		}

		// 开始创建文件
		if err := os.Mkdir(filepath.Join(pFolder, folder), 0700); err != nil {
			return err
		}
		funcs := template.FuncMap{"title": strings.Title}
		{
			//  contract.go
			file := filepath.Join(pFolder, folder, "contract.go")
			f, err := os.Create(file)
			if err != nil {
				return errors.Cause(err)
			}

			t := template.Must(template.New("contract").Funcs(funcs).Parse(contractTmp))
			if err := t.Execute(f, name); err != nil {
				return errors.Cause(err)
			}
		}
		{
			//  provider.go
			file := filepath.Join(pFolder, folder, "provider.go")
			f, err := os.Create(file)
			if err != nil {
				return err
			}
			t := template.Must(template.New("provider").Funcs(funcs).Parse(providerTmp))
			if err := t.Execute(f, name); err != nil {
				return err
			}
		}
		{
			//  service.go
			file := filepath.Join(pFolder, folder, "service.go")
			f, err := os.Create(file)
			if err != nil {
				return err
			}
			t := template.Must(template.New("service").Funcs(funcs).Parse(serviceTmp))
			if err := t.Execute(f, name); err != nil {
				return err
			}
		}
		fmt.Println("create provider success, folder path:", filepath.Join(pFolder, folder))
		fmt.Println("please remember add provider to kernel")
		return nil
	},
}

var contractTmp string = `package {{.}}

const {{.|title}}Key = "{{.}}"

type Service interface {
	// define some method here ...
}
`

var providerTmp string = `package {{.}}

import (
	"hade/framework"
)

type {{.|title}}Provider struct {
	framework.ServiceProvider

	c framework.Container
}

func (sp *{{.|title}}Provider) Name() string {
	return {{.|title}}Key
}

func (sp *{{.|title}}Provider) Register(c framework.Container) framework.NewInstance {
	return New{{.|title}}Service
}

func (sp *{{.|title}}Provider) IsDefer() bool {
	return false
}

func (sp *{{.|title}}Provider) Params() []interface{} {
	return []interface{}{sp.c}
}

func (sp *{{.|title}}Provider) Boot(c framework.Container) error {
	sp.c = c
	return nil
}

`

var serviceTmp string = `package {{.}}

import "hade/framework"

type {{.|title}}Service struct {
	container framework.Container
}

func New{{.|title}}Service(params ...interface{}) (interface{}, error) {
	container := params[0].(framework.Container)
	return &{{.|title}}Service{container: container}, nil
}
`

// TODO: provider 增加版本信息

var providerDocCommand = &cobra.Command{
	Use:   "doc",
	Short: "show detail of one provider",
	RunE: func(c *cobra.Command, args []string) error {
		return nil
	},
}

func providerDoc(provider framework.ServiceProvider) string {

	// hade内置pkg: hade/framework/provider/app
	// 自定义的pkg: hade/framework/provider/demo
	pkgPath := reflect.TypeOf(provider).Elem().PkgPath()

	// 获取对应的contract文件地址
	return pkgPath

	// 解析文件

	// 判断其中的接口文件

	// 生成文档
}
