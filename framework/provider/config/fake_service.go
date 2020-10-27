package config

import (
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

type FakeConfig struct {
	confMaps map[string]interface{}
}

func NewFakeConfig(params ...interface{}) (interface{}, error) {
	name := params[0].(string)
	bf := params[1].([]byte)
	c := map[string]interface{}{}
	if err := yaml.Unmarshal(bf, &c); err != nil {
		return nil, err
	}
	return &FakeConfig{
		confMaps: map[string]interface{}{name: c},
	}, nil
}

func (conf *FakeConfig) find(key string) interface{} {
	return searchMap(conf.confMaps, strings.Split(key, "."))
}

// IsExist check setting is exist
func (conf *FakeConfig) IsExist(key string) bool {
	return conf.find(key) != nil
}

// Get a new interface
func (conf *FakeConfig) Get(key string) interface{} {
	return conf.find(key)
}

// GetBool get bool type
func (conf *FakeConfig) GetBool(key string) bool {
	return cast.ToBool(conf.find(key))
}

// GetInt get Int type
func (conf *FakeConfig) GetInt(key string) int {
	return cast.ToInt(conf.find(key))
}

// GetFloat64 get float64
func (conf *FakeConfig) GetFloat64(key string) float64 {
	return cast.ToFloat64(conf.find(key))
}

// GetTime get time type
func (conf *FakeConfig) GetTime(key string) time.Time {
	return cast.ToTime(conf.find(key))
}

// GetString get string typen
func (conf *FakeConfig) GetString(key string) string {
	return cast.ToString(conf.find(key))
}

// GetIntSlice get int slice type
func (conf *FakeConfig) GetIntSlice(key string) []int {
	return cast.ToIntSlice(conf.find(key))
}

// GetStringSlice get string slice type
func (conf *FakeConfig) GetStringSlice(key string) []string {
	return cast.ToStringSlice(conf.find(key))
}

// GetStringMap get map which key is string, value is interface
func (conf *FakeConfig) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(conf.find(key))
}

// GetStringMapString get map which key is string, value is string
func (conf *FakeConfig) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(conf.find(key))
}

// GetStringMapStringSlice get map which key is string, value is string slice
func (conf *FakeConfig) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(conf.find(key))
}

// Load a config to a struct, val should be an pointer
func (conf *FakeConfig) Load(key string, val interface{}) error {
	return mapstructure.Decode(conf.find(key), val)
}
