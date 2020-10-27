package services

import (
	"os"
	"path/filepath"

	"hade/framework"
	"hade/framework/contract"

	"github.com/pkg/errors"
)

type HadeSingleLog struct {
	HadeLog

	folder string
	file   string
	fd     *os.File
}

// NewHadeSingleLog params sequence: level, ctxFielder, Formatter, map[string]interface(folder/file)
func NewHadeSingleLog(params ...interface{}) (interface{}, error) {
	level := params[0].(contract.LogLevel)
	ctxFielder := params[1].(contract.CtxFielder)
	formatter := params[2].(contract.Formatter)
	configs := params[3].(map[string]interface{})

	c := params[4].(framework.Container)

	log := &HadeSingleLog{}
	log.SetLevel(level)
	log.SetCxtFielder(ctxFielder)
	log.SetFormatter(formatter)
	log.folder = configs["folder"].(string) //must contain
	log.file = configs["file"].(string)     // must contain

	fd, err := os.OpenFile(filepath.Join(log.folder, log.file), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, errors.Wrap(err, "open log file err")
	}

	log.SetOutput(fd)
	log.c = c

	return log, nil
}

func (l *HadeSingleLog) SetFile(file string) {
	l.file = file
}

func (l *HadeSingleLog) SetFolder(folder string) {
	l.folder = folder
}
