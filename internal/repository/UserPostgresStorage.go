package repository

import (
	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type UserPostgresStorage struct {
	db *PostgresConnection
}

func MakeUserPostgresStorage(pc *PostgresConnection) *UserPostgresStorage {

	return &UserPostgresStorage{
		db: pc,
	}
}

func (ps *UserPostgresStorage) GetUser(login string) *models.User {

}

func (ps *UserPostgresStorage) RegisterUser(user models.User) error {

}

func (ps *UserPostgresStorage) LoginUser(user models.User) error {

}
