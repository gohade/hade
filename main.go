package main

import (
	"hade/app/console"
	"hade/app/http"
	"hade/app/provider"
	"hade/framework"
	"hade/framework/provider/app"
	"hade/framework/provider/config"
	"hade/framework/provider/env"
	"hade/framework/provider/id"
	"hade/framework/provider/kernel"
	"hade/framework/provider/log"
	"hade/framework/provider/ssh"
	"hade/framework/util"
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
