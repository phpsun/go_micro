package common

import (
	"common/discovery"
	"common/util"
)

func CommonDummy() {
	discovery.Init("")
	util.NewMysql(nil)
	util.NewRedisClient(nil)
	util.NewHttpClient()
}
