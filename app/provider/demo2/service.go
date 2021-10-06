package demo2

import "github.com/gohade/hade/framework"

type Demo2Service struct {
	container framework.Container
}

func NewDemo2Service(params ...interface{}) (interface{}, error) {
	container := params[0].(framework.Container)
	return &Demo2Service{container: container}, nil
}

func (s *Demo2Service) Foo() string {
    return ""
}
