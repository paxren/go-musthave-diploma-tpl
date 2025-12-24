package handler

import (
	"net/http"
	"strings"

	"github.com/paxren/go-musthave-diploma-tpl/internal/auth"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

type authorizer struct {
	userRepo   repository.UsersBase
	jwtService *auth.JWTService
}

func MakeAuthorizer(userRepo repository.UsersBase, jwtService *auth.JWTService) *authorizer {

	return &authorizer{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (auth *authorizer) AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {
	authFn := func(res http.ResponseWriter, req *http.Request) {

		// Извлекаем токен из заголовка Authorization
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(res, "отсутствует заголовок авторизации", http.StatusUnauthorized)
			return
		}

		var tokenString string

		// Проверяем формат заголовка: Bearer <token> или просто <token>
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
			// Формат Bearer <token>
			tokenString = tokenParts[1]
		} else if len(tokenParts) == 1 {
			// Формат просто <token> (обратная совместимость)
			tokenString = tokenParts[0]
		} else {
			http.Error(res, "неверный формат заголовка авторизации", http.StatusUnauthorized)
			return
		}

		// Валидируем JWT токен
		claims, err := auth.jwtService.ValidateToken(tokenString)
		if err != nil {
			http.Error(res, "невалидный токен", http.StatusUnauthorized)
			return
		}

		// Получаем пользователя из базы данных для проверки существования
		user := auth.userRepo.GetUser(claims.Login)
		if user == nil {
			http.Error(res, "пользователь не найден", http.StatusUnauthorized)
			return
		}

		// Проверяем, что ID пользователя совпадает с ID в токене
		if user.UserID == nil || *user.UserID != claims.UserID {
			http.Error(res, "несоответствие данных пользователя", http.StatusUnauthorized)
			return
		}

		// Добавляем пользователя в контекст запроса
		ctx := SetUserContext(req.Context(), user)
		req = req.WithContext(ctx)

		h.ServeHTTP(res, req)

	}

	return http.HandlerFunc(authFn)
}
