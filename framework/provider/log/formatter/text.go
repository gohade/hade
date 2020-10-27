package formatter

import (
	"bytes"
	"fmt"
	"time"

	"hade/framework/contract"
)

func TextFormatter(level contract.LogLevel, t time.Time, msg string, fields map[string]interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte(msg))

	prefix := Prefix(level)

	bf.WriteString(prefix)
	bf.WriteByte(' ')

	ts := t.Format(time.RFC3339)
	bf.WriteString(ts)
	bf.WriteByte(' ')

	bf.WriteString(msg)
	bf.WriteByte(' ')

	bf.WriteString(fmt.Sprint(fields))
	return bf.Bytes(), nil
}
