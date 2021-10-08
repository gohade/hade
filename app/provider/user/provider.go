package user

import (
	"github.com/gohade/hade/framework"
)

type UserProvider struct {
	framework.ServiceProvider

	c framework.Container
}

func (sp *UserProvider) Name() string {
	return UserKey
}

func (sp *UserProvider) Register(c framework.Container) framework.NewInstance {
	return NewUserService
}

func (sp *UserProvider) IsDefer() bool {
	return false
}

func (sp *UserProvider) Params(c framework.Container) []interface{} {
	return []interface{}{c}
}

func (sp *UserProvider) Boot(c framework.Container) error {
	return nil
}

