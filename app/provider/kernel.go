package provider

import (
	"github.com/gohade/hade/app/provider/demo"
	"github.com/gohade/hade/framework"
)

// customer provider
func RegisterCustomProvider(c framework.Container) {
	c.Bind(&demo.DemoProvider{})
}
