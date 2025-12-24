package handler

import (
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

// Handler основная структура обработчика
type Handler struct {
	userRepo  repository.UsersBase
	orderRepo repository.OrderBase
}

// NewHandler конструктор обработчика
func NewHandler(users repository.UsersBase, orders repository.OrderBase) *Handler {
	return &Handler{
		userRepo:  users,
		orderRepo: orders,
	}
}
