package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hade/framework"
	"hade/framework/contract"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type HadeConfig struct {
	c framework.Container

	folder   string
	env      string
	keyDelim string

	envMaps  map[string]string // envmap
	confMaps map[string]interface{}
	confRaws map[string][]byte
}

func NewHadeConfig(params ...interface{}) (interface{}, error) {
	if len(params) != 4 {
		return nil, errors.New("NewHadeConfig params error")
	}

	folder := params[0].(string)
	envMaps := params[1].(map[string]string)
	env := params[2].(string)

	c := params[3].(framework.Container)

	envFolder := filepath.Join(folder, env)
	// check folder exist
	if _, err := os.Stat(envFolder); os.IsNotExist(err) {
		return nil, errors.New("folder " + envFolder + " not exist: " + err.Error())
	}

	hadeConf := &HadeConfig{
		c:        c,
		folder:   folder,
		env:      env,
		envMaps:  envMaps,
		confMaps: map[string]interface{}{},
		confRaws: map[string][]byte{},
		keyDelim: ".",
	}

	// read all yml/yaml files in folder
	files, err := ioutil.ReadDir(envFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, file := range files {
		s := strings.Split(file.Name(), ".")
		if len(s) == 2 && (s[1] == "yaml" || s[1] == "yml") {
			name := s[0]

			// read file bytes
			bf, err := ioutil.ReadFile(filepath.Join(envFolder, file.Name()))
			if err != nil {
				continue
			}
			hadeConf.confRaws[name] = bf
			// do replace
			bf = replace(bf, envMaps)
			// parse yaml
			c := map[string]interface{}{}
			if err := yaml.Unmarshal(bf, &c); err != nil {
				continue
			}
			hadeConf.confMaps[name] = c
		}
	}

	// init app path
	if hadeConf.IsExist("app.path") && c.IsBind(contract.AppKey) {
		appPaths := hadeConf.GetStringMapString("app.path")
		appService := c.MustMake(contract.AppKey).(contract.App)
		appService.LoadAppConfig(appPaths)
	}
	return hadeConf, nil
}

func replace(content []byte, maps map[string]string) []byte {
	if maps == nil {
		return content
	}

	for key, val := range maps {
		reKey := "env(" + key + ")"
		content = bytes.ReplaceAll(content, []byte(reKey), []byte(val))
	}

	return content
}

func searchMap(source map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return source
	}

	next, ok := source[path[0]]
	if ok {
		// Fast path
		if len(path) == 1 {
			return next
		}

		// Nested case
		switch next.(type) {
		case map[interface{}]interface{}:
			return searchMap(cast.ToStringMap(next), path[1:])
		case map[string]interface{}:
			// Type assertion is safe here since it is only reached
			// if the type of `next` is the same as the type being asserted
			return searchMap(next.(map[string]interface{}), path[1:])
		default:
			// got a value but nested key expected, return "nil" for not found
			return nil
		}
	}
	return nil
}

func (conf *HadeConfig) find(key string) interface{} {
	return searchMap(conf.confMaps, strings.Split(key, conf.keyDelim))
}

// IsExist check setting is exist
func (conf *HadeConfig) IsExist(key string) bool {
	return conf.find(key) != nil
}

// Get a new interface
func (conf *HadeConfig) Get(key string) interface{} {
	return conf.find(key)
}

// GetBool get bool type
func (conf *HadeConfig) GetBool(key string) bool {
	return cast.ToBool(conf.find(key))
}

// GetInt get Int type
func (conf *HadeConfig) GetInt(key string) int {
	return cast.ToInt(conf.find(key))
}

// GetFloat64 get float64
func (conf *HadeConfig) GetFloat64(key string) float64 {
	return cast.ToFloat64(conf.find(key))
}

// GetTime get time type
func (conf *HadeConfig) GetTime(key string) time.Time {
	return cast.ToTime(conf.find(key))
}

// GetString get string typen
func (conf *HadeConfig) GetString(key string) string {
	return cast.ToString(conf.find(key))
}

// GetIntSlice get int slice type
func (conf *HadeConfig) GetIntSlice(key string) []int {
	return cast.ToIntSlice(conf.find(key))
}

// GetStringSlice get string slice type
func (conf *HadeConfig) GetStringSlice(key string) []string {
	return cast.ToStringSlice(conf.find(key))
}

// GetStringMap get map which key is string, value is interface
func (conf *HadeConfig) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(conf.find(key))
}

// GetStringMapString get map which key is string, value is string
func (conf *HadeConfig) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(conf.find(key))
}

// GetStringMapStringSlice get map which key is string, value is string slice
func (conf *HadeConfig) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(conf.find(key))
}

// Load a config to a struct, val should be an pointer
func (conf *HadeConfig) Load(key string, val interface{}) error {
	return mapstructure.Decode(conf.find(key), val)
}
