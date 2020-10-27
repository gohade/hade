package contract

import (
	"github.com/gohade/hade/framework/gin"
)

const KernelKey = "hade:kernel"

type Kernel interface {
	HttpEngine() *gin.Engine
}
