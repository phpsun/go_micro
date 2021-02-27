package main

import (
	"common/discovery"
	"common/tlog"
	"common/util"
	"config_server/logic"
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

func main() {
	var c logic.Config
	if !util.NewConfig("./config-dev.toml", &c) {
		return
	}
	tlog.Init(c.Log)

	if c.EtcdEnv == "" {
		c.EtcdEnv = c.Env
	}
	discovery.Init(c.Etcd...)

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())

	var err error
	if err = logic.NewServer(&c); err == nil {
		if err = discovery.RegisterConfigServer(c.EtcdEnv, logic.ThisServer); err == nil {
			fmt.Println(util.FormatFullTime(time.Now()), "running ...")
			discovery.WaitForClose()
		}
		logic.DestroyServer()
	}
	if err != nil {
		discovery.Close()
		fmt.Println(err)
	}

	tlog.Close()
}
