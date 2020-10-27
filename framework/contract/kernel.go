package contract

import (
	"hade/framework/gin"
)

const KernelKey = "hade:kernel"

type Kernel interface {
	HttpEngine() *gin.Engine
}
