package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
	"github.com/paxren/go-musthave-diploma-tpl/internal/money"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

// GetBalance обрабатывает получение текущего баланса пользователя
func (h Handler) GetBalance(res http.ResponseWriter, req *http.Request) {

	// Получаем пользователя из контекста
	user, err := GetUserFromContext(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	balance, err := h.orderRepo.GetBalance(*user)
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

	// Получаем пользователя из контекста
	user, err := GetUserFromContext(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// Конвертируем сумму из рублей в копейки для хранения в БД с корректным округлением
	sumInKopecks := money.RublesToKopecks(withdrawReq.Sum)

	// Создаем заказ на списание
	withdrawOrder := *models.MakeWithdraw(*user, withdrawReq.Order, sumInKopecks)

	// Добавляем заказ в базу данных
	err = h.orderRepo.AddOrder(*user, withdrawOrder)
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
	// Получаем пользователя из контекста
	user, err := GetUserFromContext(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// Получаем историю выводов
	withdrawals, err := h.orderRepo.GetWithdrawals(*user)
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
