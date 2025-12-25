# План реализации взаимодействия с системой расчёта баллов (accrual)

## Обзор задачи
Необходимо реализовать модуль взаимодействия между приложением Gophermart и системой расчёта баллов лояльности (accrual). Система accrual работает на порту 8081 и предоставляет API для получения информации о начислениях за заказы.

## Архитектура решения

### 1. Структура взаимодействия
- Реализовать периодический опрос статусов заказов в accrual системе с интервалом 5 секунд
- Обновлять статусы заказов и начислять баллы в базе данных Gophermart
- Обрабатывать ошибки и ограничения частоты запросов к accrual системе

### 2. Компоненты системы

#### 2.1. Модели данных для взаимодействия с accrual
- Создать структуру `AccrualOrderResponse` для парсинга ответов от accrual системы
- Добавить константы для статусов accrual системы (REGISTERED, INVALID, PROCESSING, PROCESSED)

#### 2.2. HTTP-клиент для взаимодействия с accrual
- Реализовать `AccrualClient` с методами:
  - `GetOrderInfo(orderNumber string) (*AccrualOrderResponse, error)`
- Добавить обработку HTTP-кодов ответа:
  - 200 - успешный ответ
  - 204 - заказ не найден
  - 429 - превышен лимит запросов (с обработкой Retry-After)
  - 500 - внутренняя ошибка сервера

#### 2.3. Реализация недостающих handlers
- Реализовать `WithdrawBalance(res http.ResponseWriter, req *http.Request)` для списания баллов
- Реализовать `GetWithdrawals(res http.ResponseWriter, req *http.Request)` для получения истории выводов
- Добавить валидацию номера заказа по алгоритму Луна в `WithdrawBalance`
- Реализовать проверку достаточности средств перед списанием

#### 2.4. Расширение репозитория заказов
- Добавить метод `GetOrdersWithStatuses(statuses []string) ([]models.Order, error)` для получения заказов с определёнными статусами
- Добавить метод `UpdateOrderStatusAndValue(orderID, status string, value uint64) error` для обновления статуса и начисления баллов
- Добавить метод `GetWithdrawals(user models.User) ([]models.Order, error)` для получения истории выводов

#### 2.5. Сервис опроса статусов
- Реализовать `AccrualPollingService` с функционалом:
  - Периодический опрос заказов со статусами NEW и PROCESSING
  - Обновление статусов в базе данных
  - Обработка ошибок и повторных попыток
  - Управление частотой запросов для соблюдения лимитов accrual системы

#### 2.6. Интеграция в основное приложение
- Добавить запуск сервиса опроса в `main.go`
- Реализовать graceful shutdown для корректного завершения работы сервиса

## Детальная реализация

### Шаг 1: Модели данных для accrual
```go
// AccrualOrderResponse представляет ответ от accrual системы
type AccrualOrderResponse struct {
    Order   string  `json:"order"`
    Status  string  `json:"status"`
    Accrual *uint64 `json:"accrual,omitempty"`
}

// Константы для статусов accrual системы
const (
    AccrualStatusRegistered = "REGISTERED"
    AccrualStatusInvalid    = "INVALID"
    AccrualStatusProcessing = "PROCESSING"
    AccrualStatusProcessed  = "PROCESSED"
)
```

### Шаг 2: HTTP-клиент для accrual
```go
type AccrualClient struct {
    baseURL    string
    httpClient *http.Client
    logger     *log.Logger
}

func NewAccrualClient(baseURL string) *AccrualClient
func (c *AccrualClient) GetOrderInfo(orderNumber string) (*AccrualOrderResponse, error)
```

### Шаг 3: Реализация недостающих handlers
```go
// В handler.go добавить:
func (h Handler) WithdrawBalance(res http.ResponseWriter, req *http.Request)
func (h Handler) GetWithdrawals(res http.ResponseWriter, req *http.Request)

// Структура для запроса списания:
type WithdrawRequest struct {
    Order string `json:"order"`
    Sum   uint64 `json:"sum"`
}

// Структура для ответа истории выводов:
type WithdrawResponse struct {
    Order       string `json:"order"`
    Sum         uint64 `json:"sum"`
    ProcessedAt string `json:"processed_at"`
}
```

### Шаг 4: Расширение репозитория
```go
// В OrderBase интерфейс добавить:
GetOrdersWithStatuses(statuses []string) ([]models.Order, error)
UpdateOrderStatusAndValue(orderID, status string, value uint64) error
GetWithdrawals(user models.User) ([]models.Order, error)

// Реализация в OrderPostgresStorage:
func (st *OrderPostgresStorage) GetOrdersWithStatuses(statuses []string) ([]models.Order, error)
func (st *OrderPostgresStorage) UpdateOrderStatusAndValue(orderID, status string, value uint64) error
func (st *OrderPostgresStorage) GetWithdrawals(user models.User) ([]models.Order, error)
```

### Шаг 5: Сервис опроса статусов
```go
type AccrualPollingService struct {
    accrualClient *AccrualClient
    orderRepo     repository.OrderBase
    logger        *log.Logger
    ticker        *time.Ticker
    done          chan bool
}

func NewAccrualPollingService(accrualClient *AccrualClient, orderRepo repository.OrderBase) *AccrualPollingService
func (s *AccrualPollingService) Start()
func (s *AccrualPollingService) Stop()
func (s *AccrualPollingService) pollOrders()
func (s *AccrualPollingService) processOrder(order models.Order)
```

### Шаг 6: Интеграция в приложение
```go
// В main.go:
// 1. Создать accrual клиент
accrualClient := services.NewAccrualClient(serverConfig.GetAccrualSystemURL())

// 2. Создать сервис опроса
pollingService := services.NewAccrualPollingService(accrualClient, ordersStorage)

// 3. Раскомментировать и исправить эндпоинты
r.Post(`/api/user/balance/withdraw`, authMidl.AuthMiddleware(handlerv.WithdrawBalance))
r.Get(`/api/user/withdrawals`, authMidl.AuthMiddleware(handlerv.GetWithdrawals))

// 4. Запустить сервис опроса
pollingService.Start()
defer pollingService.Stop()
```

## Обработка ошибок и ограничения

### 1. Ограничение частоты запросов
- Реализовать rate limiting при получении 429 ответа от accrual системы
- Учитывать заголовок Retry-After для определения времени ожидания
- Использовать экспоненциальный backoff для повторных попыток

### 2. Обработка ошибок
- Логировать все ошибки взаимодействия с accrual системой
- Продолжать работу сервиса при временных сбоях
- Реализовать механизм повторных попыток для сетевых ошибок

### 3. Логирование
- Логировать начало и завершение опроса заказов
- Логировать обновления статусов заказов
- Логировать ошибки взаимодействия с accrual системой

## Тестирование
1. Проверить корректность обновления статусов заказов
2. Проверить начисление баллов при изменении статуса на PROCESSED
3. Проверить обработку ошибок и ограничений частоты запросов
4. Проверить работу сервиса при недоступности accrual системы

## Последовательность реализации
1. Изучить структуру базы данных и миграции для понимания схемы хранения заказов
2. Разработать структуру для взаимодействия с accrual системой (модели данных, HTTP-клиент)
3. Реализовать HTTP-клиент для запросов к accrual системе
4. Добавить методы в репозиторий для получения заказов со статусами NEW и PROCESSING
5. Реализовать сервис для периодического опроса статусов заказов
6. Добавить логику обновления статусов и начисления баллов в базе данных
7. Интегрировать сервис опроса в основное приложение
8. Добавить обработку ошибок и ограничение частоты запросов (rate limiting)
9. Реализовать логирование для отслеживания взаимодействия с accrual системой
10. Протестировать работу системы с accrual

## Запуск и тестирование приложения

### Запуск Gophermart
Для запуска приложения Gophermart необходимо выполнить команду:
```bash
DATABASE_URI="host=localhost user=dbtest1 password=dbtest1 dbname=dbtest1 sslmode=disable" ./gophermart
```

### Примеры запросов для тестирования

#### Аутентификация пользователя
```bash
curl -v -X POST 'http://localhost:8080/api/user/login' \
  -H "Content-Type: application/json" \
  -d '{"login": "user","password": "123"}'
```

#### Добавление заказа
```bash
curl -v -X POST 'http://localhost:8080/api/user/orders' \
  -H "Content-Type: text/plain" \
  -H "Authorization: user" \
  -d '12345678903'
```

#### Получение списка заказов
```bash
curl -v -X GET 'http://localhost:8080/api/user/orders' \
  -H "Authorization: user"
```

#### Получение баланса
```bash
curl -v -X GET 'http://localhost:8080/api/user/balance' \
  -H "Authorization: user"
```

### Тестирование взаимодействия с accrual
1. Запустить accrual систему на порту 8081 (по умолчанию)
2. Запустить Gophermart с указанной выше командой
3. Выполнить аутентификацию пользователя
4. Добавить заказ с валидным номером (проверяется алгоритмом Луна)
5. Наблюдать за обновлением статуса заказа через GET /api/user/orders
6. Проверить начисление баллов через GET /api/user/balance

## Примечания
- Опрос заказов будет выполняться каждые 5 секунд для быстрого обновления статусов
- Будут обрабатываться только заказы со статусами NEW и PROCESSING
- При получении финальных статусов (INVALID, PROCESSED) заказы больше не будут опрашиваться
- Начисление баллов будет происходить автоматически при изменении статуса на PROCESSED
- Система accrual должна быть запущена на порту 8081 (настраивается через флаг -r или переменную окружения ACCRUAL_SYSTEM_ADDRESS)