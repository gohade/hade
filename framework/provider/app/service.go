package app

type HadeApp struct {
	basePath string

	storagePath  string
	logPath      string
	cachePath    string
	coveragePath string
	pidPath      string
}

func NewHadeApp(params ...interface{}) (interface{}, error) {
	var basePath string
	if len(params) == 1 {
		basePath = params[0].(string)
	}
	return &HadeApp{basePath: basePath}, nil
}

// application version
func (app *HadeApp) Version() string {
	return "0.2"
}

// base path which is the base folder
func (app *HadeApp) BasePath() string {
	return app.basePath
}

// base path which is the base folder
func (app *HadeApp) AppPath() string {
	return app.BasePath() + "app/"
}

func (app *HadeApp) HttpPath() string {
	return app.AppPath() + "http/"
}

func (app *HadeApp) SwaggerPath() string {
	return app.HttpPath() + "swagger/"
}

func (app *HadeApp) ConsolePath() string {
	return app.AppPath() + "console/"
}

// config folder which contains config
func (app *HadeApp) ConfigPath() string {
	return app.BasePath() + "config/"
}

// environmentPath which contain .env
func (app *HadeApp) EnvironmentPath() string {
	return app.BasePath()
}

// storagePath define storage folder
func (app *HadeApp) StoragePath() string {
	if app.storagePath != "" {
		return app.storagePath
	}
	return app.BasePath() + "storage/"
}

// logPath define logPath
func (app *HadeApp) LogPath() string {
	if app.logPath != "" {
		return app.logPath
	}
	return app.StoragePath() + "log/"
}

func (app *HadeApp) PidPath() string {
	if app.pidPath != "" {
		return app.pidPath
	}
	return app.StoragePath() + "pid/"
}

func (app *HadeApp) CachePath() string {
	if app.cachePath != "" {
		return app.cachePath
	}
	return app.StoragePath() + "cache/"
}

func (app *HadeApp) LoadAppConfig(kv map[string]string) {
	if v, ok := kv["storage"]; ok {
		app.storagePath = v
	}
	if v, ok := kv["log"]; ok {
		app.logPath = v
	}
	if v, ok := kv["pid"]; ok {
		app.pidPath = v
	}
	if v, ok := kv["cache"]; ok {
		app.cachePath = v
	}
}
