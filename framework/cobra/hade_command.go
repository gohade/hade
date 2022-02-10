package cobra

import (
	"log"

	"github.com/gohade/hade/framework"
)

// SetContainer 设置服务容器
func (c *Command) SetContainer(container framework.Container) {
	c.container = container
}

// GetContainer 获取容器
func (c *Command) GetContainer() framework.Container {
	return c.Root().container
}

// CronSpec 保存Cron命令的信息，用于展示
type CronSpec struct {
	Type        string
	Cmd         *Command
	Spec        string
	ServiceName string
}

func (c *Command) SetParantNull() {
	c.parent = nil
}

// AddCronCommand 是用来创建一个Cron任务的
func (c *Command) AddCronCommand(spec string, cmd *Command) {
	// cron结构是挂载在根Command上的
	root := c.Root()
	// 增加说明信息
	root.CronSpecs = append(root.CronSpecs, CronSpec{
		Type: "normal-cron",
		Cmd:  cmd,
		Spec: spec,
	})

	// 增加调用函数
	root.Cron.AddFunc(spec, func() {

		// 制作一个rootCommand，必须放在这个里面做复制，否则会产生竞态
		var cronCmd Command
		ctx := root.Context()
		cronCmd = *cmd
		cronCmd.args = []string{}
		cronCmd.SetParantNull()
		cronCmd.SetContainer(root.GetContainer())

		// 如果后续的command出现panic，这里要捕获
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
			}
		}()

		err := cronCmd.ExecuteContext(ctx)
		if err != nil {
			// 打印出err信息
			log.Println(err)
		}
	})
}
