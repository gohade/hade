package app

import (
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/util"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

// HadeApp 代表hade框架的App实现
type HadeApp struct {
	container  framework.Container // 服务容器
	baseFolder string              // 基础路径
	appId      string              // 表示当前这个app的唯一id, 可以用于分布式锁等

	configMap map[string]string // 配置加载
	envMap    map[string]string // 环境变量加载
	argsMap   map[string]string // 参数加载
}

// AppID 表示这个App的唯一ID
func (app HadeApp) AppID() string {
	return app.appId
}

// Version 实现版本
func (app HadeApp) Version() string {
	return HadeVersion
}

// BaseFolder 表示基础目录，可以代表开发场景的目录，也可以代表运行时候的目录
func (app HadeApp) BaseFolder() string {
	if app.baseFolder != "" {
		return app.baseFolder
	}
	baseFolder := app.getConfigBySequence("base_folder", "BASE_FOLDER", "app.path.base_folder")
	if baseFolder != "" {
		return baseFolder
	}

	// 如果参数也没有，使用默认的当前路径
	return util.GetExecDirectory()
}

// ConfigFolder  表示配置文件地址
func (app HadeApp) ConfigFolder() string {
	val := app.getConfigBySequence("config_folder", "CONFIG_FOLDER", "app.path.config_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "config")
}

// LogFolder 表示日志存放地址
func (app HadeApp) LogFolder() string {
	val := app.getConfigBySequence("log_folder", "LOG_FOLDER", "app.path.log_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.StorageFolder(), "log")
}

func (app HadeApp) HttpFolder() string {
	val := app.getConfigBySequence("http_folder", "HTTP_FOLDER", "app.path.http_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "app", "http")
}

func (app HadeApp) ConsoleFolder() string {
	val := app.getConfigBySequence("console_folder", "CONSOLE_FOLDER", "app.path.console_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "app", "console")
}

func (app HadeApp) StorageFolder() string {
	val := app.getConfigBySequence("storage_folder", "STORAGE_FOLDER", "app.path.storage_folder")
	if val != "" {
		return val
	}

	return filepath.Join(app.BaseFolder(), "storage")
}

// ProviderFolder 定义业务自己的服务提供者地址
func (app HadeApp) ProviderFolder() string {
	val := app.getConfigBySequence("provider_folder", "PROVIDER_FOLDER", "app.path.provider_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "app", "provider")
}

// MiddlewareFolder 定义业务自己定义的中间件
func (app HadeApp) MiddlewareFolder() string {
	val := app.getConfigBySequence("middleware_folder", "MIDDLEWARE_FOLDER", "app.path.middleware_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.HttpFolder(), "middleware")
}

// CommandFolder 定义业务定义的命令
func (app HadeApp) CommandFolder() string {
	val := app.getConfigBySequence("command_folder", "COMMAND_FOLDER", "app.path.command_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.ConsoleFolder(), "command")
}

// RuntimeFolder 定义业务的运行中间态信息
func (app HadeApp) RuntimeFolder() string {
	val := app.getConfigBySequence("runtime_folder", "RUNTIME_FOLDER", "app.path.runtime_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.StorageFolder(), "runtime")
}

// TestFolder 定义测试需要的信息
func (app HadeApp) TestFolder() string {
	val := app.getConfigBySequence("test_folder", "TEST_FOLDER", "app.path.test_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "test")
}

// DeployFolder 定义测试需要的信息
func (app HadeApp) DeployFolder() string {
	val := app.getConfigBySequence("deploy_folder", "DEPLOY_FOLDER", "app.path.deploy_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "deploy")
}

// AppFolder 代表app目录
func (app *HadeApp) AppFolder() string {
	val := app.getConfigBySequence("app_folder", "APP_FOLDER", "app.path.app_folder")
	if val != "" {
		return val
	}
	return filepath.Join(app.BaseFolder(), "app")
}

// NewHadeApp 初始化HadeApp
func NewHadeApp(params ...interface{}) (interface{}, error) {
	if len(params) != 2 {
		return nil, errors.New("param error")
	}

	// 有两个参数，一个是容器，一个是baseFolder
	container := params[0].(framework.Container)
	baseFolder := params[1].(string)

	appId := uuid.New().String()
	configMap := map[string]string{}
	hadeApp := &HadeApp{baseFolder: baseFolder, container: container, appId: appId, configMap: configMap}
	_ = hadeApp.loadEnvMaps()
	_ = hadeApp.loadArgsMaps()
	return hadeApp, nil
}

// GetConfigByDefault 默认获取配置项的方法
// 配置优先级：参数>环境变量>配置文件
func (app *HadeApp) getConfigBySequence(argsKey string, envKey string, configKey string) string {
	if app.argsMap != nil {
		if val, ok := app.argsMap[argsKey]; ok {
			return val
		}
	}

	if app.envMap != nil {
		if val, ok := app.envMap[envKey]; ok {
			return val
		}
	}

	if app.configMap != nil {
		if val, ok := app.configMap[configKey]; ok {
			return val
		}
	}
	return ""
}

func (app *HadeApp) loadEnvMaps() error {
	if app.envMap == nil {
		app.envMap = map[string]string{}
	}
	// 读取环境变量
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		app.envMap[pair[0]] = pair[1]
	}
	return nil
}

func (app *HadeApp) loadArgsMaps() error {
	if app.argsMap == nil {
		app.argsMap = map[string]string{}
	}
	// load args, must be format : --key=value
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--") {
			pair := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
			if len(pair) == 2 {
				app.argsMap[pair[0]] = pair[1]
			}
		}
	}
	return nil
}

// LoadAppConfig 加载配置map
func (app *HadeApp) LoadAppConfig(kv map[string]string) {
	for key, val := range kv {
		app.configMap[key] = val
	}
}
