package repository

import (
	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type MemStorage struct {
	users map[string]models.User
}

func MakeMemStorage() *MemStorage {

	return &MemStorage{
		users: make(map[string]models.User),
	}
}

func (m *MemStorage) GetUser(login string) *models.User {

	v, ok := m.users[login]

	if !ok {
		return nil
	}

	return &v
}

func (m *MemStorage) RegisterUser(user models.User) error {
	if m.GetUser(user.Login) != nil {
		return ErrUserExist
	}
	m.users[user.Login] = user

	//todo доработать крайние случаи

	return nil

}

func (m *MemStorage) LoginUser(user models.User) error {

	dbUser := m.GetUser(user.Login)
	if dbUser == nil || (dbUser.Login != user.Login) || (dbUser.Password != user.Password) {
		return ErrBadLogin
	}

	return nil
}
