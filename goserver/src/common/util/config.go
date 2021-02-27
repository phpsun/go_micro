package util

import (
	"flag"
	"fmt"

	"github.com/BurntSushi/toml"
)

func NewConfig(defaultPath string, v interface{}) bool {
	confName := flag.String("config", defaultPath, "config file")
	flag.Parse()

	fmt.Printf("Load config file: %s\n", *confName)
	_, err := toml.DecodeFile(*confName, v)
	if err != nil {
		fmt.Println("config file load failed:", *confName, err)
		return false
	}
	return true
}
