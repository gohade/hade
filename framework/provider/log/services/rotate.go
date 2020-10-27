package services

import (
	"fmt"
	"path/filepath"
	"time"

	"hade/framework"
	"hade/framework/contract"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/pkg/errors"
)

type HadeRotateLog struct {
	HadeLog

	folder     string
	file       string
	maxFiles   int
	dateFormat string
}

func NewHadeRotateLog(params ...interface{}) (interface{}, error) {
	level := params[0].(contract.LogLevel)
	ctxFielder := params[1].(contract.CtxFielder)
	formatter := params[2].(contract.Formatter)
	configs := params[3].(map[string]interface{})
	c := params[4].(framework.Container)

	folder := configs["folder"].(string)
	file := configs["file"].(string)
	maxFiles := configs["max_files"].(int)
	dateFormat := configs["date_format"].(string)

	log := &HadeRotateLog{}
	log.SetLevel(level)
	log.SetCxtFielder(ctxFielder)
	log.SetFormatter(formatter)
	log.SetFile(file)
	log.SetFolder(folder)
	log.SetMaxFiles(maxFiles)
	log.SetDateFormat(dateFormat)

	w, err := rotatelogs.New(fmt.Sprintf("%s.%s", filepath.Join(log.folder, log.file), log.dateFormat),
		rotatelogs.WithLinkName(filepath.Join(log.folder, log.file)),
		rotatelogs.WithMaxAge(24*time.Hour*time.Duration(log.maxFiles)),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithLocation(time.Local))
	if err != nil {
		return nil, errors.Wrap(err, "new rotatelogs error")
	}
	log.SetOutput(w)
	log.c = c
	return log, nil
}

func (l *HadeRotateLog) SetFolder(folder string) {
	l.folder = folder
}

func (l *HadeRotateLog) SetFile(file string) {
	l.file = file
}

func (l *HadeRotateLog) SetMaxFiles(maxFiles int) {
	l.maxFiles = maxFiles
}

func (l *HadeRotateLog) SetDateFormat(dateFormat string) {
	l.dateFormat = dateFormat
}
