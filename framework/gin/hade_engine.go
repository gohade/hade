package gin

import "github.com/gohade/hade/framework"

// SetContainer 为Engine设置container
func (engine *Engine) SetContainer(container framework.Container) {
	engine.container = container
}

// GetContainer 从Engine中获取container
func (engine *Engine) GetContainer() framework.Container {
	return engine.container
}

// engine实现container的绑定封装
func (engine *Engine) Bind(provider framework.ServiceProvider) error {
	return engine.container.Bind(provider)
}

// IsBind 关键字凭证是否已经绑定服务提供者
func (engine *Engine) IsBind(key string) bool {
	return engine.container.IsBind(key)
}
