package handler

import (
	"github.com/paxren/go-musthave-diploma-tpl/internal/auth"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

// Handler основная структура обработчика
type Handler struct {
	userRepo   repository.UsersBase
	orderRepo  repository.OrderBase
	jwtService *auth.JWTService
}

// NewHandler конструктор обработчика
func NewHandler(users repository.UsersBase, orders repository.OrderBase, jwtService *auth.JWTService) *Handler {
	return &Handler{
		userRepo:   users,
		orderRepo:  orders,
		jwtService: jwtService,
	}
}
