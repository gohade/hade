package xerror

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	baseErr := New(OK, "success")
	assert.Equal(t, baseErr.Error(), GetErr(OK).Error())
	assert.Equal(t, baseErr.GetCode(), GetErr(OK).GetCode())
	assert.Equal(t, baseErr.GetOriginCode(), GetErr(OK).GetOriginCode())
	assert.Equal(t, baseErr.GetMsg(), GetErr(OK).GetMsg())
	assert.Equal(t, baseErr.GetData(), GetErr(OK).GetData())

	baseErr1 := New(100, "xxxxxxx").WithMsg("wsx")
	assert.Equal(t, baseErr1.GetCode(), int32(100))
	assert.Equal(t, baseErr1.GetMsg(), "wsx")

	baseErr2 := baseErr1.WithData("111111111")
	assert.Equal(t, baseErr2.GetData(), "111111111")

	baseErr3 := GetErr(Code(99999))
	assert.Equal(t, baseErr3.Error(), GetErr(UnknownCode).Error())
	assert.Equal(t, baseErr3.GetCode(), GetErr(UnknownCode).GetCode())
	assert.Equal(t, baseErr3.GetOriginCode(), GetErr(UnknownCode).GetOriginCode())
}
