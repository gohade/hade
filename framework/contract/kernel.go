package contract

import (
	"net/http"
)

const KernelKey = "hade:kernel"

// Kernel 接口提供框架最核心的结构
type Kernel interface {
	// HttpEngine 提供gin的Engine结构
	HttpEngine() http.Handler
}
