package handler

import (
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

// AddOrder обрабатывает добавление нового заказа
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

	// Получаем пользователя из контекста
	user, err := GetUserFromContext(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	err = h.orderRepo.AddOrder(*user, *models.MakeNewOrder(*user, orderString))
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

// GetOrders обрабатывает получение списка заказов пользователя
func (h Handler) GetOrders(res http.ResponseWriter, req *http.Request) {

	// Получаем пользователя из контекста
	user, err := GetUserFromContext(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	orders, err := h.orderRepo.GetOrders(*user, models.OrderType)
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
