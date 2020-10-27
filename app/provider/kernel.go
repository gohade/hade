package provider

import (
	"hade/app/provider/demo"
	"hade/framework"
)

// customer provider
func RegisterCustomProvider(c framework.Container) {
	c.Bind(&demo.DemoProvider{}, true)
}
