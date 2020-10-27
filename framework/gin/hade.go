package gin

import "github.com/gohade/hade/framework"

// Hade framework add functions

func (engine *Engine) SetContainer(container *framework.HadeContainer) {
	engine.container = container
}

func (engine *Engine) Container() *framework.HadeContainer {
	return engine.container
}
