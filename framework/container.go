package framework

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// Container is a core struct which store provider and instance
type Container interface {
	// Bind bind a service provider
	Bind(provider ServiceProvider, isSingleton bool) error
	// Singlton is Bind a singlton service provider
	Singleton(provider ServiceProvider) error
	// IsBind check a service provider has been bind
	IsBind(key string) bool

	// Make to get a service provider
	Make(key string) (interface{}, error)
	// MustMake to get a service provider which will not return error
	// if error，panic it
	// If you use it, make sure it will not panic, or you can cover it
	MustMake(key string) interface{}
	// MakeNew to new a service
	// The service Must not be singlton, it with params to make a new service
	MakeNew(key string, params []interface{}) (interface{}, error)
}

// HadeContainer is instance of Container
type HadeContainer struct {
	Container
	providers    []ServiceProvider
	instances    map[string]interface{}
	methods      map[string]NewInstance
	isSingletons map[string]bool

	lock sync.RWMutex
}

// NewHadeContainer is new instance
func NewHadeContainer() *HadeContainer {
	return &HadeContainer{
		providers:    []ServiceProvider{},
		instances:    map[string]interface{}{},
		methods:      map[string]NewInstance{},
		isSingletons: map[string]bool{},
		lock:         sync.RWMutex{},
	}
}

func (hade *HadeContainer) GetProviders() []ServiceProvider {
	return hade.providers
}

func (hade *HadeContainer) PrintList() []string {
	ret := []string{}
	for _, provider := range hade.providers {
		name := provider.Name()
		// register := provider.Register(hade)
		// funcName := reflect.TypeOf(register).Name()

		line := fmt.Sprint(name)
		ret = append(ret, line)
	}
	return ret
}

// Bind make relationship between provider and contract
func (hade *HadeContainer) Bind(provider ServiceProvider, isSingleton bool) error {
	hade.lock.RLock()
	defer hade.lock.RUnlock()
	key := provider.Name()

	hade.providers = append(hade.providers, provider)
	hade.isSingletons[key] = isSingleton
	hade.methods[key] = provider.Register(hade)

	// if provider is not defer
	if provider.IsDefer() == false {
		if err := provider.Boot(hade); err != nil {
			return err
		}
		params := provider.Params()
		method := hade.methods[key]
		instance, err := method(params...)
		if err != nil {
			return errors.New(err.Error())
		}
		if isSingleton == true {
			hade.instances[key] = instance
		}
	}
	return nil
}

// Singleton make provider be Singleton, instance once
func (hade *HadeContainer) Singleton(provider ServiceProvider) error {
	return hade.Bind(provider, true)
}

func (hade *HadeContainer) IsBind(key string) bool {
	return hade.findServiceProvider(key) != nil
}

func (hade *HadeContainer) findServiceProvider(key string) ServiceProvider {
	for _, sp := range hade.providers {
		if sp.Name() == key {
			return sp
		}
	}
	return nil
}

func (hade *HadeContainer) Make(key string) (interface{}, error) {
	return hade.make(key, nil, false)
}

func (hade *HadeContainer) MustMake(key string) interface{} {
	serv, err := hade.make(key, nil, false)
	if err != nil {
		panic(err)
	}
	return serv
}

func (hade *HadeContainer) MakeNew(key string, params []interface{}) (interface{}, error) {
	return hade.make(key, params, true)
}

func (hade *HadeContainer) make(key string, params []interface{}, isNew bool) (interface{}, error) {
	// check has Register
	if hade.findServiceProvider(key) == nil {
		return nil, errors.New("contract " + key + " have not register")
	}

	// if isNew, call boot
	if isNew == false {
		// check instance
		if ins, ok := hade.instances[key]; ok {
			return ins, nil
		}
	}

	// is not instance
	method := hade.methods[key] // must ok
	prov := hade.findServiceProvider(key)
	isSingle := hade.isSingletons[key]
	if err := prov.Boot(hade); err != nil {
		return nil, err
	}
	if params == nil {
		params = prov.Params()
	}
	ins, err := method(params...)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	if isSingle {
		hade.instances[key] = ins
		return ins, nil
	}
	return ins, nil
}
