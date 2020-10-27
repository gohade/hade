package main

import (
	"github.com/gohade/hade/app/console"
	"github.com/gohade/hade/app/http"
	"github.com/gohade/hade/app/provider"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/provider/app"
	"github.com/gohade/hade/framework/provider/config"
	"github.com/gohade/hade/framework/provider/env"
	"github.com/gohade/hade/framework/provider/id"
	"github.com/gohade/hade/framework/provider/kernel"
	"github.com/gohade/hade/framework/provider/log"
	"github.com/gohade/hade/framework/provider/ssh"
	"github.com/gohade/hade/framework/util"
)

func main() {
	container := framework.NewHadeContainer()

	basePath := util.GetExecDirectory()
	container.Singleton(&app.HadeAppProvider{BasePath: basePath})
	container.Singleton(&env.HadeEnvProvider{})
	container.Singleton(&config.HadeConfigProvider{})
	container.Singleton(&log.HadeLogServiceProvider{})
	container.Singleton(&id.HadeIDProvider{})
	container.Singleton(&ssh.HadeSSHProvider{})

	if engine, err := http.NewHttpEngine(); err == nil {
		container.Singleton(&kernel.HadeKernelProvider{HttpEngine: engine})
	}

	// custom register
	provider.RegisterCustomProvider(container)

	console.RunCommand(container)
}
