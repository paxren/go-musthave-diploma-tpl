package handler

import (
	"context"
	"errors"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// userContextKey используется для хранения пользователя в контексте
type userContextKey string

const UserKey userContextKey = "user"

// SetUserContext добавляет пользователя в контекст запроса
func SetUserContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

// GetUserFromContext извлекает пользователя из контекста запроса
func GetUserFromContext(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(UserKey).(*models.User)
	if !ok || user == nil {
		return nil, errors.New("пользователь не найден в контексте")
	}
	return user, nil
}
