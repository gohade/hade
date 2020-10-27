package contract

// AppKey is the key in container
const AppKey = "hade:app"

// App define application structure
type App interface {
	// application version
	Version() string
	// base path which is the base folder
	BasePath() string
	// app path which is the app folder
	AppPath() string
	// app/http
	HttpPath() string
	// app/http/swagger
	SwaggerPath() string
	// app/console
	ConsolePath() string
	// config folder which contains config
	ConfigPath() string
	// environmentPath which contain .env
	EnvironmentPath() string
	// storagePath define storage folder
	StoragePath() string
	// logPath define logPath
	LogPath() string

	PidPath() string

	CachePath() string
	// load config
	LoadAppConfig(map[string]string)
}
