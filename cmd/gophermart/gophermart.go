package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/paxren/go-musthave-diploma-tpl/internal/config"
	"github.com/paxren/go-musthave-diploma-tpl/internal/handler"
	"github.com/paxren/go-musthave-diploma-tpl/internal/logger"
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
	//обработка сигтерм, по статье https://habr.com/ru/articles/908344/
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	finish := make([]func() error, 0, 1)

	serverConfig.Parse()

	fmt.Println()
	fmt.Println(serverConfig)

	// Создаем и устанавливаем логгер по умолчанию
	appLogger := logger.New()
	logger.SetDefault(appLogger)

	postgresCon, err := repository.MakePostgresStorage(serverConfig.DatabaseURI)
	if err != nil {
		appLogger.Error("PostgreSQL не инициализирована", "error", err)
		panic("посгря не инициализирована")
	}
	finish = append(finish, postgresCon.Close)

	usersStorage := repository.MakeUserPostgresStorage(postgresCon)
	ordersStorage := repository.MakeOrderPostgresStorage(postgresCon)
	authMidl := handler.MakeAuthorizer(usersStorage)
	handlerv := handler.NewHandler(usersStorage, ordersStorage)
	r := chi.NewRouter()

	// Создаем клиент для взаимодействия с accrual системой
	accrualClient := services.NewAccrualClient(serverConfig.AccrualSystemAddress)
	accrualClient.SetLogger(appLogger)

	// Создаем сервис опроса статусов заказов
	pollingService := services.NewAccrualPollingService(accrualClient, ordersStorage)
	pollingService.SetLogger(appLogger)

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

	go func() {
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			appLogger.Error("Ошибка при запуске сервера", "error", err)
			panic(err)
		}

	}()

	appLogger.Info("Запуск сервера", "address", serverConfig.RunAddress.String())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Error("Ошибка при запуске сервера", "error", err)
	}

	//обработка сигтерм TODO добработать или переработать после понимания контекста и др
	<-rootCtx.Done()
	appLogger.Info("Получен сигнал завершения, остановка сервера")
	stop()

	if err := server.Shutdown(context.Background()); err != nil {
		appLogger.Error("Ошибка при остановке сервера", "error", err)
	}

	for _, f := range finish {
		if err := f(); err != nil {
			appLogger.Error("Ошибка при выполнении финализатора", "error", err)
		}
	}

	appLogger.Info("Сервер успешно остановлен")
}
