package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
	"github.com/paxren/go-musthave-diploma-tpl/internal/money"
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

// AccrualOrderResponse представляет ответ от accrual системы
type AccrualOrderResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// Константы для статусов accrual системы
const (
	AccrualStatusRegistered = "REGISTERED"
	AccrualStatusInvalid    = "INVALID"
	AccrualStatusProcessing = "PROCESSING"
	AccrualStatusProcessed  = "PROCESSED"
)

// AccrualClient представляет клиент для взаимодействия с системой расчёта баллов
type AccrualClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAccrualClient создает новый экземпляр AccrualClient
func NewAccrualClient(baseURL string) *AccrualClient {
	return &AccrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})), // По умолчанию без логирования
	}
}

// SetLogger устанавливает slog логгер для клиента
func (c *AccrualClient) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// GetOrderInfo получает информацию о заказе из системы accrual с механизмом повторных попыток
func (c *AccrualClient) GetOrderInfo(orderNumber string) (*AccrualOrderResponse, error) {
	const maxRetries = 3
	const baseRetryDelay = 1 * time.Second

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Экспоненциальный backoff с jitter
			delay := baseRetryDelay * time.Duration(1<<uint(attempt-1))
			c.logger.Info("Повторная попытка запроса заказа",
				"order_number", orderNumber,
				"attempt", attempt+1,
				"max_attempts", maxRetries,
				"delay", delay)
			time.Sleep(delay)
		}

		response, err := c.getOrderInfoOnce(orderNumber)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Если это ошибка ограничения частоты запросов, пробуем повторить
		if isRateLimitError(err) {
			// Извлекаем время ожидания из ошибки, если возможно
			if retryAfter := extractRetryAfter(err); retryAfter > 0 {
				c.logger.Info("Ожидание перед повторной попыткой из-за ограничения частоты",
					"retry_after_seconds", retryAfter)
				time.Sleep(time.Duration(retryAfter) * time.Second)
				continue
			}
		}

		// Для других ошибок не повторяем, кроме последней попытки
		if attempt < maxRetries-1 && !isRetryableError(err) {
			break
		}
	}

	return nil, lastErr
}

// getOrderInfoOnce выполняет один запрос к accrual системе
func (c *AccrualClient) getOrderInfoOnce(orderNumber string) (*AccrualOrderResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	c.logger.Info("Запрос информации о заказе из accrual системы",
		"order_number", orderNumber,
		"url", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Error("Ошибка при выполнении запроса к accrual системе",
			"error", err,
			"order_number", orderNumber)
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrualResponse AccrualOrderResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrualResponse); err != nil {
			c.logger.Error("Ошибка при декодировании ответа от accrual системы",
				"error", err,
				"order_number", orderNumber)
			return nil, fmt.Errorf("ошибка при декодировании ответа: %w", err)
		}

		c.logger.Info("Получен ответ от accrual системы",
			"order_number", orderNumber,
			"status", accrualResponse.Status,
			"accrual", accrualResponse.Accrual)

		return &accrualResponse, nil

	case http.StatusNoContent:
		c.logger.Info("Заказ не найден в accrual системе", "order_number", orderNumber)
		return nil, fmt.Errorf("заказ не найден в accrual системе")

	case http.StatusTooManyRequests:
		// Получаем заголовок Retry-After если он есть
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				c.logger.Warn("Превышен лимит запросов к accrual системе",
					"retry_after_seconds", seconds,
					"order_number", orderNumber)
				return nil, fmt.Errorf("превышен лимит запросов, повтор через %d секунд", seconds)
			}
		}
		c.logger.Warn("Превышен лимит запросов к accrual системе",
			"order_number", orderNumber)
		return nil, fmt.Errorf("превышен лимит запросов к accrual системе")

	case http.StatusInternalServerError:
		c.logger.Error("Внутренняя ошибка сервера accrual системы",
			"order_number", orderNumber,
			"status_code", resp.StatusCode)
		return nil, fmt.Errorf("внутренняя ошибка сервера accrual системы")

	default:
		c.logger.Error("Неожиданный статус код от accrual системы",
			"order_number", orderNumber,
			"status_code", resp.StatusCode)
		return nil, fmt.Errorf("неожиданный статус код: %d", resp.StatusCode)
	}
}

// isRateLimitError проверяет, является ли ошибка ошибкой ограничения частоты запросов
func isRateLimitError(err error) bool {
	return err != nil && (containsString(err.Error(), "превышен лимит запросов") ||
		containsString(err.Error(), "too many requests"))
}

// isRetryableError проверяет, можно ли повторить запрос при данной ошибке
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return containsString(errStr, "ошибка при выполнении запроса") ||
		containsString(errStr, "внутренняя ошибка сервера") ||
		isRateLimitError(err)
}

// extractRetryAfter извлекает время ожидания из ошибки ограничения частоты запросов
func extractRetryAfter(err error) int {
	if err == nil {
		return 0
	}

	errStr := err.Error()
	// Ищем паттерн "повтор через X секунд"
	if idx := findSubstring(errStr, "повтор через "); idx != -1 {
		remaining := errStr[idx+len("повтор через "):]
		if spaceIdx := findSubstring(remaining, " "); spaceIdx != -1 {
			secondsStr := remaining[:spaceIdx]
			if seconds, parseErr := strconv.Atoi(secondsStr); parseErr == nil {
				return seconds
			}
		}
	}

	return 0
}

// containsString проверяет, содержит ли строка подстроку (без учета регистра)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr) != -1))
}

// findSubstring находит индекс подстроки в строке
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// AccrualPollingService представляет сервис для периодического опроса статусов заказов
type AccrualPollingService struct {
	accrualClient *AccrualClient
	orderRepo     repository.OrderBase
	logger        *slog.Logger
	ticker        *time.Ticker
	done          chan bool
}

// NewAccrualPollingService создает новый экземпляр AccrualPollingService
func NewAccrualPollingService(accrualClient *AccrualClient, orderRepo repository.OrderBase) *AccrualPollingService {
	return &AccrualPollingService{
		accrualClient: accrualClient,
		orderRepo:     orderRepo,
		logger:        slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})), // По умолчанию без логирования
		done:          make(chan bool),
	}
}

// SetLogger устанавливает slog логгер для сервиса опроса
func (s *AccrualPollingService) SetLogger(logger *slog.Logger) {
	s.logger = logger
	s.accrualClient.SetLogger(logger)
}

// Start запускает сервис опроса статусов
func (s *AccrualPollingService) Start() {
	s.logger.Info("Запуск сервиса опроса статусов заказов")

	// Создаем тикер с интервалом 1 секунды
	s.ticker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.pollOrders()
			case <-s.done:
				s.logger.Info("Остановка сервиса опроса статусов заказов")
				return
			}
		}
	}()
}

// Stop останавливает сервис опроса статусов
func (s *AccrualPollingService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Отправляем сигнал о завершении работы
	select {
	case s.done <- true:
	default:
		// Канал уже закрыт или сигнал уже отправлен
	}
}

// pollOrders выполняет опрос заказов со статусами NEW и PROCESSING
func (s *AccrualPollingService) pollOrders() {
	s.logger.Debug("Начало опроса статусов заказов")

	// Получаем заказы со статусами NEW и PROCESSING
	statuses := []string{models.OrderStatusNew, models.OrderStatusProcessing}
	orders, err := s.orderRepo.GetOrdersWithStatuses(statuses)
	if err != nil {
		s.logger.Error("Ошибка при получении заказов для опроса", "error", err)
		return
	}

	if len(orders) == 0 {
		s.logger.Debug("Нет заказов для опроса")
		return
	}

	s.logger.Info("Найдено заказов для опроса", "count", len(orders))

	// Обрабатываем каждый заказ
	for _, order := range orders {
		s.processOrder(order)
	}

	s.logger.Debug("Завершение опроса статусов заказов")
}

// processOrder обрабатывает один заказ
func (s *AccrualPollingService) processOrder(order models.Order) {
	s.logger.Info("Обработка заказа",
		"order_id", order.OrderID,
		"status", order.Status)

	// Получаем информацию о заказе из accrual системы
	accrualResponse, err := s.accrualClient.GetOrderInfo(order.OrderID)
	if err != nil {
		s.logger.Error("Ошибка при получении информации о заказе",
			"error", err,
			"order_id", order.OrderID)
		return
	}

	// Проверяем, изменился ли статус
	if accrualResponse.Status == order.Status {
		s.logger.Debug("Статус заказа не изменился",
			"order_id", order.OrderID,
			"status", order.Status)
		return
	}

	s.logger.Info("Статус заказа изменился",
		"order_id", order.OrderID,
		"old_status", order.Status,
		"new_status", accrualResponse.Status)

	// Определяем значение для начисления (конвертируем из рублей в копейки)
	var accrualValue uint64 = 0
	if accrualResponse.Accrual != nil {
		// Accrual возвращает сумму в рублях, конвертируем в копейки для хранения в БД
		// Используем безопасную конвертацию с проверкой на переполнение
		var err error
		accrualValue, err = money.AccrualToKopecks(*accrualResponse.Accrual)
		if err != nil {
			s.logger.Error("Ошибка при конвертации суммы начисления",
				"error", err,
				"order_id", order.OrderID,
				"accrual", *accrualResponse.Accrual)
			return
		}
	}

	// Обновляем статус и значение заказа в базе данных
	err = s.orderRepo.UpdateOrderStatusAndValue(order.OrderID, accrualResponse.Status, accrualValue)
	if err != nil {
		s.logger.Error("Ошибка при обновлении статуса заказа",
			"error", err,
			"order_id", order.OrderID,
			"status", accrualResponse.Status,
			"accrual_value", accrualValue)
		return
	}

	s.logger.Info("Заказ успешно обновлен",
		"order_id", order.OrderID,
		"status", accrualResponse.Status,
		"accrual_value", accrualValue)
}
