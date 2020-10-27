package demo

import (
	"fmt"

	"hade/framework"
)

type DemoServiceProvider struct {
	C map[string]string
	framework.ServiceProvider
}

func (sp *DemoServiceProvider) Name() string {
	return "demo"
}

func (sp *DemoServiceProvider) Register(c framework.Container) framework.NewInstance {
	return NewDemoService
}

func (sp *DemoServiceProvider) IsDefer() bool {
	return true
}

func (sp *DemoServiceProvider) Params() []interface{} {
	return []interface{}{sp.C}
}

func (sp *DemoServiceProvider) Boot(c framework.Container) error {
	fmt.Println("demo service boot")
	return nil
}
