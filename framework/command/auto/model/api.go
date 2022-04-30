package model

type Api struct {
	Name     string     `yaml:"name"`
	Path     string     `yaml:"path"`
	Method   string     `yaml:"method"`
	Params   []Params   `yaml:"params"`
	Workflow []Workflow `yaml:"workflow"`
}

type Params struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Validate string `yaml:"validate"`
}

type Workflow struct {
	Type     string `yaml:"type"`
	Database string `yaml:"database"`
	Sql      string `yaml:"sql"`
	Output   Output `yaml:"output"`
	Func     string `yaml:"func"`
}

type Output struct {
	Type   string   `yaml:"type"`
	Fields []Fields `yaml:"fields"`
}

type Fields struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Adapter string `yaml:"adapter"`
}
