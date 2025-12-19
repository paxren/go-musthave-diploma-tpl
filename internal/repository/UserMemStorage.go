package repository

import (
	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type UserMemStorage struct {
	users map[string]models.User
}

func MakeUserMemStorage() *UserMemStorage {

	return &UserMemStorage{
		users: make(map[string]models.User),
	}
}

func (m *UserMemStorage) GetUser(login string) *models.User {

	v, ok := m.users[login]

	if !ok {
		return nil
	}

	return &v
}

func (m *UserMemStorage) RegisterUser(user models.User) error {
	if m.GetUser(user.Login) != nil {
		return ErrUserExist
	}
	id := uint64(len(m.users) + 1)
	user.ID = &id
	m.users[user.Login] = user

	//todo доработать крайние случаи

	return nil

}

func (m *UserMemStorage) LoginUser(user models.User) error {

	dbUser := m.GetUser(user.Login)
	if dbUser == nil || (dbUser.Login != user.Login) || (dbUser.Password != user.Password) {
		return ErrBadLogin
	}

	return nil
}
