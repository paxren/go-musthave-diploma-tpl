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

	postgresCon, err := repository.MakePostgresStorage(serverConfig.DatabaseURI)
	if err != nil {
		panic("посгря не инициализирована")
	}

	usersStorage := repository.MakeUserPostgresStorage(postgresCon)
	ordersStorage := repository.MakeOrderPostgresStorage(postgresCon)
	authMidl := handler.MakeAuthorizer(usersStorage)
	handlerv := handler.NewHandler(usersStorage, ordersStorage)
	r := chi.NewRouter()

	r.Post(`/api/user/register`, handlerv.RegisterUser)
	r.Post(`/api/user/login`, handlerv.LoginUser)
	r.Post(`/api/user/orders`, authMidl.AuthMiddleware(handlerv.AddOrder))
	r.Get(`/api/user/orders`, authMidl.AuthMiddleware(handlerv.GetOrders))
	r.Get(`/api/user/balance`, authMidl.AuthMiddleware(handlerv.GetBalance))
	// r.Post(`/api/user/balance/withdraw`, authMidl.AuthMiddleware(handlerv.LoginUser))
	// r.Get(`/api/user/withdrawals`, authMidl.AuthMiddleware(handlerv.LoginUser))

	server := &http.Server{
		Addr:    serverConfig.RunAddress.String(),
		Handler: r,
	}

	server.ListenAndServe()
}
