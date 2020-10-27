package id

import (
	"github.com/rs/xid"
)

type HadeIDService struct {
}

func NewHadeIDService(params ...interface{}) (interface{}, error) {
	return &HadeIDService{}, nil
}

func (s *HadeIDService) NewID() string {
	return xid.New().String()
}
