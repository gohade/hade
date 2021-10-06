package demo2

import (
	"github.com/gohade/hade/framework"
)

type Demo2Provider struct {
	framework.ServiceProvider

	c framework.Container
}

func (sp *Demo2Provider) Name() string {
	return Demo2Key
}

func (sp *Demo2Provider) Register(c framework.Container) framework.NewInstance {
	return NewDemo2Service
}

func (sp *Demo2Provider) IsDefer() bool {
	return false
}

func (sp *Demo2Provider) Params(c framework.Container) []interface{} {
	return []interface{}{c}
}

func (sp *Demo2Provider) Boot(c framework.Container) error {
	return nil
}

