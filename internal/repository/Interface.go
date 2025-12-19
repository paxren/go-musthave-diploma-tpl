package repository

import (
	"errors"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

var (
	ErrUserExist = errors.New("пользаватель уже существует")
	ErrBadLogin  = errors.New("не авторизован. неверный логин и/или пароль")

	ErrOrderExistThisUser    = errors.New("заказ уже существует у этого пользователя")
	ErrOrderExistAnotherUser = errors.New("заказ уже существует у другого пользователя")

	ErrOrderType = errors.New("неизвестный тип заказа")

	ErrIncafitionFunds = errors.New("недостаточно средств для списания")

	ErrBadOrderId = errors.New("плохой номер заказа (не луноподходящий)")
)

type UsersBase interface {
	GetUser(name string) *models.User
	RegisterUser(user models.User) error
	LoginUser(user models.User) error
}

type OrderBase interface {
	AddOrder(user models.User, order models.Order) error
	GetOrders(user models.User, orderType string) ([]models.Order, error)
	GetBalance(user models.User) (*models.Balance, error)
}
