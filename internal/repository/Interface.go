package repository

import (
	"errors"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

var (
	ErrUserExist = errors.New("пользаватель уже существует")
	ErrBadLogin  = errors.New("не авторизован. неверный логин и/или пароль")
)

type UsersBase interface {
	GetUser(name string) *models.User
	RegisterUser(user models.User) error
	LoginUser(user models.User) error
}
