package util

import (
	"context"
	"fmt"

	"hade/framework"
	"hade/framework/cobra"
)

type ContainerKey string

const containerKey = ContainerKey("container")

func RegiestContainer(c framework.Container, cmd *cobra.Command) context.Context {
	ctx := context.WithValue(context.Background(), containerKey, c)
	// cmd.SetContext(ctx)
	return ctx
}

func GetContainer(cmd *cobra.Command) framework.Container {
	val := cmd.Context().Value(containerKey)
	if val == nil {
		fmt.Println("val is nil, register")
		container := framework.NewHadeContainer()
		RegiestContainer(container, cmd)
		return container
	}
	return val.(framework.Container)
}
