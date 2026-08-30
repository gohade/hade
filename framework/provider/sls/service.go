package sls

import (
	"github.com/aliyun/aliyun-log-go-sdk/producer"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
)

type HadeSLSService struct {
	producerInstance *producer.Producer
	project          string // SLS 日志项目
	logstore         string // SLS 日志库
}

func NewHadeSLSService(params ...interface{}) (interface{}, error) {
	container := params[0].(framework.Container)
	configService := container.MustMake(contract.ConfigKey).(contract.Config)

	endpoint := configService.GetString("app.sls.endpoint")
	accessKeyID := configService.GetString("app.sls.access_key_id")
	accessKeySecret := configService.GetString("app.sls.access_key_secret")
	project := configService.GetString("app.sls.project")
	logstore := configService.GetString("app.sls.logstore")

	producerConfig := producer.GetDefaultProducerConfig()
	producerConfig.Endpoint = endpoint
	producerConfig.AccessKeyID = accessKeyID
	producerConfig.AccessKeySecret = accessKeySecret

	producerInstance := producer.InitProducer(producerConfig)

	producerInstance.Start()
	return &HadeSLSService{
		producerInstance: producerInstance,
		project:          project,
		logstore:         logstore,
	}, nil
}

func (sls *HadeSLSService) GetSLSInstance() (*producer.Producer, error) {
	return sls.producerInstance, nil
}

func (sls *HadeSLSService) GetProject() (string, error) {
	return sls.project, nil
}

func (sls *HadeSLSService) GetLogstore() (string, error) {
	return sls.logstore, nil
}
