package defs

import "time"

const (
	DaySeconds  = 86400
	DayDuration = DaySeconds * time.Second

	EnvProd            = "prod"
	EnvTest            = "test"
	EnvDev             = "dev"
	ErrUnknownResponse = 112

	ErrOk = 0

	ErrCommon              = 101
	ErrInternal            = 500
	ErrNetworkAuthRequired = 511

	PlatformWeb       = 1
	PlatformWebMobile = 2
	PlatformWebPc     = 3
	PlatformWebWap    = 4
	PlatformAndroid   = 5
	PlatformIos       = 6

	CtxKeyHashKey = "hash_key"
)

var SessionCookie = map[string]string{
	EnvProd: "con_session",
	EnvTest: "con_session_t1",
	EnvDev:  "con_session_dev",
}
