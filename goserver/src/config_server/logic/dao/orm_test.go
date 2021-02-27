package dao

import (
	"common/util"
	"config_server/logic"
	"fmt"
	"testing"
)

func TestGetData(t *testing.T) {
	var c logic.Config
	if !util.NewConfig("../config-dev.toml", &c) {
		return
	}
	logic.NewServer(&c)

	var users []*ConfRegions
	logic.ThisServer.OshopSystem.Find(&users)
	fmt.Println(users[1])
}
