package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
	"github.com/paxren/go-musthave-diploma-tpl/internal/money"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

type Handler struct {
	userRepo  repository.UsersBase
	orderRepo repository.OrderBase
}

type BalanceExport struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type OrderExport struct {
	OrderID string   `json:"number"`
	User    string   `json:"-"`
	Type    string   `json:"-"`
	Status  string   `json:"status"`
	Date    string   `json:"uploaded_at"`
	Value   *float64 `json:"accrual,omitempty"`
}

func NewHandler(users repository.UsersBase, orders repository.OrderBase) *Handler {
	return &Handler{
		userRepo:  users,
		orderRepo: orders,
	}
}

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

func (h Handler) AddOrder(res http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(res, "нужен плейн текст", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Преобразуем байты в строку
	orderString := string(body)
	// Здесь можно добавить валидацию номера заказа
	if orderString == "" {
		http.Error(res, "пустой номер заказа", http.StatusBadRequest)
		return
	}

	userHeader := req.Header.Get("User")
	userDB := h.userRepo.GetUser(userHeader)
	err = h.orderRepo.AddOrder(*userDB, *models.MakeNewOrder(*userDB, orderString))
	if err != nil {
		if errors.Is(err, repository.ErrOrderExistThisUser) {
			res.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, repository.ErrOrderExistAnotherUser) {
			http.Error(res, "номер заказа уже был загружен другим пользователем", http.StatusConflict)
			return
		}
		if errors.Is(err, repository.ErrBadOrderID) {
			http.Error(res, "плохой номер заказа, нелунуется", http.StatusUnprocessableEntity)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return

	}

	res.WriteHeader(http.StatusAccepted)
}

func (h Handler) GetOrders(res http.ResponseWriter, req *http.Request) {

	userHeader := req.Header.Get("User")
	userDB := h.userRepo.GetUser(userHeader)

	orders, err := h.orderRepo.GetOrders(*userDB, models.OrderType)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	//сразу сортировка по дате
	sort.Slice(orders, func(i, j int) bool {
		dateI, errI := time.Parse(time.RFC3339, orders[i].Date)
		dateJ, errJ := time.Parse(time.RFC3339, orders[j].Date)

		if errI != nil && errJ != nil {
			return false
		}
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}

		return dateI.After(dateJ) // от новых к старым
	})

	// Преобразуем orders в exportOrders с правильным форматированием поля accrual
	var exportOrders []OrderExport
	for _, order := range orders {
		exportOrder := OrderExport{
			OrderID: order.OrderID,
			User:    order.User,
			Type:    order.Type,
			Status:  order.Status,
			Date:    order.Date,
		}

		// Добавляем поле accrual только если значение не нулевое
		if order.Value > 0 {
			accrual := money.KopecksToRubles(order.Value)
			exportOrder.Value = &accrual
		}

		exportOrders = append(exportOrders, exportOrder)
	}

	ordersJSON, err := json.Marshal(exportOrders)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(ordersJSON)
}

func (h Handler) GetBalance(res http.ResponseWriter, req *http.Request) {

	userHeader := req.Header.Get("User")
	userDB := h.userRepo.GetUser(userHeader)

	balance, err := h.orderRepo.GetBalance(*userDB)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	exportBalance := BalanceExport{
		Current:   money.KopecksToRubles(balance.Current),
		Withdrawn: money.KopecksToRubles(balance.Withdrawn),
	}

	balanceJSON, err := json.Marshal(exportBalance)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(balanceJSON)
}

// WithdrawRequest представляет запрос на списание баллов
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// WithdrawResponse представляет ответ с информацией о выводе средств
type WithdrawResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// WithdrawBalance обрабатывает запрос на списание баллов
func (h Handler) WithdrawBalance(res http.ResponseWriter, req *http.Request) {
	// Проверяем Content-Type
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "нужен application/json", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	var withdrawReq WithdrawRequest
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Десериализуем JSON
	if err = json.Unmarshal(buf.Bytes(), &withdrawReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация номера заказа по алгоритму Луна
	if !models.LunaCheck(withdrawReq.Order) {
		http.Error(res, "неверный номер заказа", http.StatusUnprocessableEntity)
		return
	}

	// Проверяем, что сумма списания больше 0
	if withdrawReq.Sum == 0 {
		http.Error(res, "сумма списания должна быть больше 0", http.StatusBadRequest)
		return
	}

	// Получаем пользователя из заголовка
	userHeader := req.Header.Get("User")
	userDB := h.userRepo.GetUser(userHeader)
	if userDB == nil {
		http.Error(res, "пользователь не найден", http.StatusUnauthorized)
		return
	}

	// Конвертируем сумму из рублей в копейки для хранения в БД с корректным округлением
	sumInKopecks := money.RublesToKopecks(withdrawReq.Sum)

	// Создаем заказ на списание
	withdrawOrder := *models.MakeWithdraw(*userDB, withdrawReq.Order, sumInKopecks)

	// Добавляем заказ в базу данных
	err = h.orderRepo.AddOrder(*userDB, withdrawOrder)
	if err != nil {
		if errors.Is(err, repository.ErrIncafitionFunds) {
			http.Error(res, "на счету недостаточно средств", http.StatusPaymentRequired)
			return
		}
		if errors.Is(err, repository.ErrOrderExistThisUser) {
			http.Error(res, "заказ с таким номером уже существует", http.StatusConflict)
			return
		}
		if errors.Is(err, repository.ErrOrderExistAnotherUser) {
			http.Error(res, "заказ с таким номером уже существует у другого пользователя", http.StatusConflict)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// GetWithdrawals обрабатывает запрос на получение истории выводов
func (h Handler) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	// Получаем пользователя из заголовка
	userHeader := req.Header.Get("User")
	userDB := h.userRepo.GetUser(userHeader)
	if userDB == nil {
		http.Error(res, "пользователь не найден", http.StatusUnauthorized)
		return
	}

	// Получаем историю выводов
	withdrawals, err := h.orderRepo.GetWithdrawals(*userDB)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Если нет выводов, возвращаем 204
	if len(withdrawals) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	// Преобразуем в формат ответа
	var withdrawalsResponse []WithdrawResponse
	for _, withdrawal := range withdrawals {
		withdrawalsResponse = append(withdrawalsResponse, WithdrawResponse{
			Order:       withdrawal.OrderID,
			Sum:         money.KopecksToRubles(withdrawal.Value), // Конвертируем из копеек в рубли
			ProcessedAt: withdrawal.Date,
		})
	}

	// Сортируем по дате (от новых к старым)
	sort.Slice(withdrawalsResponse, func(i, j int) bool {
		dateI, errI := time.Parse(time.RFC3339, withdrawalsResponse[i].ProcessedAt)
		dateJ, errJ := time.Parse(time.RFC3339, withdrawalsResponse[j].ProcessedAt)

		if errI != nil && errJ != nil {
			return false
		}
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}

		return dateI.After(dateJ)
	})

	// Сериализуем и отправляем ответ
	withdrawalsJSON, err := json.Marshal(withdrawalsResponse)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(withdrawalsJSON)
}
