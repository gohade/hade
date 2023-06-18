package test

import (
    "github.com/gohade/hade/framework"
    "github.com/gohade/hade/framework/provider/app"
    "github.com/gohade/hade/framework/provider/env"
    "github.com/gohade/hade/framework/util"
)

const (
    BasePath = "" // 自定义
)

func GetBasePath() string {
    root, err := util.GetRootDirectory()
    if err != nil {
        return ""
    }
    return root
}

func InitBaseContainer() framework.Container {
    // 初始化服务容器
    container := framework.NewHadeContainer()
    // 绑定App服务提供者
    var basePath = BasePath
    if basePath == "" {
        basePath = GetBasePath()
    }
    container.Bind(&app.HadeAppProvider{BaseFolder: basePath})
    // 后续初始化需要绑定的服务提供者...
    container.Bind(&env.HadeTestingEnvProvider{})
    return container
}
