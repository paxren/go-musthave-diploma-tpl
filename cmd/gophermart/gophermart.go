package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/paxren/go-musthave-diploma-tpl/internal/config"
	"github.com/paxren/go-musthave-diploma-tpl/internal/handler"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
	"github.com/paxren/go-musthave-diploma-tpl/internal/services"
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

	// Создаем клиент для взаимодействия с accrual системой
	accrualClient := services.NewAccrualClient(serverConfig.GetAccrualSystemURL())

	// Создаем логгер для сервиса опроса
	logger := log.New(os.Stdout, "ACCURAL: ", log.LstdFlags)
	accrualClient.SetLogger(logger)

	// Создаем сервис опроса статусов заказов
	pollingService := services.NewAccrualPollingService(accrualClient, ordersStorage)
	pollingService.SetLogger(logger)

	// Запускаем сервис опроса
	pollingService.Start()
	defer pollingService.Stop()

	r.Post(`/api/user/register`, handlerv.RegisterUser)
	r.Post(`/api/user/login`, handlerv.LoginUser)
	r.Post(`/api/user/orders`, authMidl.AuthMiddleware(handlerv.AddOrder))
	r.Get(`/api/user/orders`, authMidl.AuthMiddleware(handlerv.GetOrders))
	r.Get(`/api/user/balance`, authMidl.AuthMiddleware(handlerv.GetBalance))
	r.Post(`/api/user/balance/withdraw`, authMidl.AuthMiddleware(handlerv.WithdrawBalance))
	r.Get(`/api/user/withdrawals`, authMidl.AuthMiddleware(handlerv.GetWithdrawals))

	server := &http.Server{
		Addr:    serverConfig.RunAddress.String(),
		Handler: r,
	}

	// Настройка graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		// Получаем сигнал завершения, останавливаем сервер
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Printf("Ошибка при остановке сервера: %v", err)
		}
	}()

	logger.Printf("Запуск сервера на адресе %s", serverConfig.RunAddress.String())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Printf("Ошибка при запуске сервера: %v", err)
	}
}
