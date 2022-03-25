package xerror

import (
	"fmt"
)

// Err struct
type err struct {
	Code Code        `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Error implements the error interface.
func (e *err) Error() string {
	return fmt.Sprintf("hadeerror: code = %v msg = %s", e.GetCode(), e.GetMsg())
}

// New returns an error object for the code, message.
func New(code Code, message string) *err {
	return &err{
		Code: code,
		Msg:  message,
		Data: struct{}{},
	}
}

// GetCode return the packaged Code
func (e *err) GetCode() int32 {
	return int32(e.Code)
}

// GetOriginCode return the origin Code
func (e *err) GetOriginCode() Code {
	return e.Code
}

// GetMsg return the Msg
func (e *err) GetMsg() string {
	return e.Msg
}

// GetData return the Data
func (e *err) GetData() interface{} {
	return e.Data
}

// WithMsg allows the programmer to override Msg and return the new Err
func (e *err) WithMsg(msg string) *err {
	return &err{
		Code: e.Code,
		Msg:  msg,
		Data: e.Data,
	}
}

// WithData allows the programmer to override Data and return the new Err
func (e *err) WithData(data interface{}) *err {
	return &err{
		Code: e.Code,
		Msg:  e.Msg,
		Data: data,
	}
}
