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

	usersStorage := repository.MakeUserMemStorage()
	ordersStorage := repository.MakeOrderMemStorage()
	handlerv := handler.NewHandler(usersStorage, ordersStorage)
	r := chi.NewRouter()

	r.Post(`/api/user/register`, handlerv.RegisterUser)
	r.Post(`/api/user/login`, handlerv.LoginUser)
	r.Post(`/api/user/orders`, handlerv.AddOrder)
	r.Get(`/api/user/orders`, handlerv.GetOrders)
	r.Get(`/api/user/balance`, handlerv.GetBalance)
	// r.Post(`/api/user/balance/withdraw`, handlerv.LoginUser)
	// r.Get(`/api/user/withdrawals`, handlerv.LoginUser)

	server := &http.Server{
		Addr:    serverConfig.RunAddress.String(),
		Handler: r,
	}

	server.ListenAndServe()
}
