package main

import (
	"api_server/logic"
	"common/discovery"
	"common/tlog"
	"common/util"
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

func main() {
	var c logic.Config
	if !util.NewConfig("./api-dev.toml", &c) {
		return
	}
	tlog.Init(c.Log)

	if c.EtcdEnv == "" {
		c.EtcdEnv = c.Env
	}
	discovery.Init(c.Etcd...)
	//go func() {
	//	tlog.Info(http.ListenAndServe("0.0.0.0:32123", nil)) //火焰图
	//}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())

	if err := logic.NewServer(&c); err == nil {
		logic.DestroyServer()
	} else {
		fmt.Println(err)
	}

	discovery.Close()
	tlog.Close()
}
