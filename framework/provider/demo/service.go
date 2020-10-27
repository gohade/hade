package demo

import (
	"errors"
	"fmt"
)

type DemoService struct {
	c map[string]string
}

func NewDemoService(params ...interface{}) (interface{}, error) {
	c := params[0].(map[string]string)
	fmt.Println("new demo service")
	return &DemoService{c: c}, errors.New("new demo service error")
}

func (s *DemoService) Get(key string) string {
	if v, ok := s.c[key]; ok {
		return v
	}
	return ""
}
