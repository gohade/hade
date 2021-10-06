package formatter

import (
	"bytes"
	"fmt"
	"time"

	"github.com/gohade/hade/framework/contract"
)

// TextFormatter 表示文本格式输出
func TextFormatter(level contract.LogLevel, t time.Time, msg string, fields map[string]interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	Separator := "\t"

	// 先输出日志级别
	prefix := Prefix(level)

	bf.WriteString(prefix)
	bf.WriteString(Separator)

	// 输出时间
	ts := t.Format(time.RFC3339)
	bf.WriteString(ts)
	bf.WriteString(Separator)

	// 输出msg
	bf.WriteString("\"")
	bf.WriteString(msg)
	bf.WriteString("\"")
	bf.WriteString(Separator)

	// 输出map
	bf.WriteString(fmt.Sprint(fields))
	return bf.Bytes(), nil
}
