package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

// readUser читает и валидирует данные пользователя из запроса
func readUser(res http.ResponseWriter, req *http.Request) (*models.User, error) {

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusResetContent)
		return nil, errors.New("нужен джейсон")
	}

	var user models.User
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return nil, err
	}
	// десериализуем JSON в Metric
	if err = json.Unmarshal(buf.Bytes(), &user); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return nil, err
	}

	if user.Login == "" || user.Password == "" {
		http.Error(res, "пустой логин и/или пароль", http.StatusBadRequest)
		return nil, errors.New("пустой логин и/или пароль")
	}

	return &user, nil
}

// RegisterUser обрабатывает регистрацию нового пользователя
func (h Handler) RegisterUser(res http.ResponseWriter, req *http.Request) {
	//_ := chi.URLParam(req, "metric_type")

	user, err := readUser(res, req)
	if err != nil {
		return
	}

	if err = h.userRepo.RegisterUser(*user); err != nil {
		if errors.Is(err, repository.ErrUserExist) {
			http.Error(res, "логин уже занят", http.StatusConflict)
		} else {
			http.Error(res, "другая ошибка при попытке зарегистрировать пользователя", http.StatusInternalServerError)
		}

		return
	}

	res.Header().Set("Authorization", user.Login)

	res.WriteHeader(http.StatusOK)

}

// LoginUser обрабатывает авторизацию пользователя
func (h Handler) LoginUser(res http.ResponseWriter, req *http.Request) {

	user, err := readUser(res, req)
	if err != nil {
		return
	}

	if err = h.userRepo.LoginUser(*user); err != nil {
		http.Error(res, "не авторизован", http.StatusUnauthorized)
		return
	}

	res.Header().Set("Authorization", user.Login)

	res.WriteHeader(http.StatusOK)
}
