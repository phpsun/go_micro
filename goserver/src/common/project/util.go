package project

import (
	"common/defs"
)

type GeneralResult struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(data interface{}) *GeneralResult {
	return &GeneralResult{
		Code:    defs.ErrOk,
		Message: "success",
		Data:    data,
	}
}

func Fail(errCode int32, errMsg string) *GeneralResult {
	return &GeneralResult{
		Code:    errCode,
		Message: errMsg,
	}
}
