package contract

import "github.com/aliyun/aliyun-log-go-sdk/producer"

const SLSKey = "hade:sls"

type SLSService interface {
	GetSLSInstance() (*producer.Producer, error)
	GetProject() (string, error)
	GetLogstore() (string, error)
}
