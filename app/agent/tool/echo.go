package tool

import (
	"context"
	"encoding/json"
)

// Echo 返回参数中的 text；缺少 text 时原样返回参数，便于安全调试。
func Echo(_ context.Context, argsJSON string) (string, error) {
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err == nil && args.Text != "" {
		return args.Text, nil
	}
	return argsJSON, nil
}
