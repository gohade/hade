package command

import (
	"fmt"
	"testing"

	"hade/app/provider/demo"
	"hade/framework/provider/app"
)

func Test_hade_providerDoc(t *testing.T) {
	appProvider := &app.HadeAppProvider{}
	got := providerDoc(appProvider)
	fmt.Println(got)
	return
}

func Test_custom_providerDoc(t *testing.T) {
	demoProvider := &demo.DemoProvider{}
	got2 := providerDoc(demoProvider)
	fmt.Println(got2)
	return
}
