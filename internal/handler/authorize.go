package handler

import (
	"net/http"

	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

type authorizer struct {
	userRepo repository.UsersBase
}

func MakeAuthorizer(userRepo repository.UsersBase) *authorizer {

	return &authorizer{
		userRepo: userRepo,
	}
}

func (auth *authorizer) AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {
	authFn := func(res http.ResponseWriter, req *http.Request) {

		authorization := req.Header.Get("Authorization")
		if authorization == "" {
			http.Error(res, "нет заголовка авторизации", http.StatusUnauthorized)
			return
		}

		//проверка авторизации
		user := auth.userRepo.GetUser(authorization)
		if user == nil {
			http.Error(res, "не авторизован", http.StatusUnauthorized)
			return
		}
		req.Header.Set("User", user.Login)

		h.ServeHTTP(res, req)

	}

	return http.HandlerFunc(authFn)
}
