package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/paxren/go-musthave-diploma-tpl/internal/config"
	"github.com/paxren/go-musthave-diploma-tpl/internal/handler"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
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

	userStorage := repository.MakeUserMemStorage()
	handlerv := handler.NewHandler(userStorage)
	r := chi.NewRouter()

	r.Post(`/api/user/register`, handlerv.RegisterUser)
	r.Post(`/api/user/login`, handlerv.LoginUser)

	server := &http.Server{
		Addr:    serverConfig.RunAddress.String(),
		Handler: r,
	}

	server.ListenAndServe()
}
