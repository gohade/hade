# hade:app

提供基础的 app 框架目录结构

``` golang
package contract

// AppKey is the key in container
const AppKey = "hade:app"

// App define application structure
type App interface {
	// application version
	Version() string
	// base path which is the base folder
	BasePath() string
	// config folder which contains config
	ConfigPath() string
	// environmentPath which contain .env
	EnvironmentPath() string
	// storagePath define storage folder
	StoragePath() string
	// logPath define logPath
	LogPath() string
	// frameworkPath define frameworkPath
	FrameworkPath() string
}

```