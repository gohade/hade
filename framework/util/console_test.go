package util

import "testing"

func TestPrettyPrint(t *testing.T) {
	type args struct {
		arr [][]string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "normal",
			args: args{
				arr: [][]string{
					{"te", "test", "sdf"},
					{"te11232", "test123123", "1232123"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrettyPrint(tt.args.arr)
		})
	}
}
