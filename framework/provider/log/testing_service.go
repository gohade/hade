package log

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/log/formatter"
	"github.com/gohade/hade/framework/provider/log/services"
	"os"
)

// HadeTestingLog 是 Env 的具体实现
type HadeTestingLog struct {
}

// NewHadeTestingLog 测试日志，直接打印到控制台
func NewHadeTestingLog(params ...interface{}) (interface{}, error) {
	log := &services.HadeConsoleLog{}

	log.SetLevel(contract.DebugLevel)
	log.SetCtxFielder(nil)
	log.SetFormatter(formatter.TextFormatter)

	// 最重要的将内容输出到控制台
	log.SetOutput(os.Stdout)
	return log, nil
}
