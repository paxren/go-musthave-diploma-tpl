package main

import (
	"fmt"

	"github.com/paxren/go-musthave-diploma-tpl/internal/config"
)

var (
	serverConfig = config.NewServerConfig()
)

func init() {
	serverConfig.Init()
}

func main() {
	serverConfig.Parse()

	fmt.Println()
	fmt.Println(serverConfig)
}
