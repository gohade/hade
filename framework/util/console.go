package util

import (
	"fmt"
)

// 美观输出数组
func PrettyPrint(arr [][]string) {
	if len(arr) == 0 {
		return
	}
	rows := len(arr)
	cols := len(arr[0])

	lens := make([][]int, rows)
	for i := 0; i < rows; i++ {
		lens[i] = make([]int, cols)
		for j := 0; j < cols; j++ {
			lens[i][j] = len(arr[i][j])
		}
	}

	colMaxs := make([]int, cols)
	for j := 0; j < cols; j++ {
		for i := 0; i < rows; i++ {
			if colMaxs[j] < lens[i][j] {
				colMaxs[j] = lens[i][j]
			}
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			fmt.Print(arr[i][j])
			padding := colMaxs[j] - lens[i][j] + 2
			for p := 0; p < padding; p++ {
				fmt.Print(" ")
			}
		}
		fmt.Print("\n")
	}
}
