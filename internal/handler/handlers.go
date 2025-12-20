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
	"github.com/paxren/go-musthave-diploma-tpl/internal/repository"
)

type Handler struct {
	userRepo  repository.UsersBase
	orderRepo repository.OrderBase

	//todo переделать!!!
	//dbConnectionString string
}

type BalanceExport struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
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
	// orderNumber, err := strconv.ParseUint(orderString, 10, 64)
	// if err != nil {
	// 	http.Error(res, "неверный формат номера заказа", http.StatusBadRequest)
	// 	return
	// }

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

	ordersJSON, err := json.Marshal(orders)
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
		Current:   (float64(balance.Current) / 100),
		Withdrawn: (float64(balance.Withdrawn) / 100),
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
	Order string `json:"order"`
	Sum   uint64 `json:"sum"`
}

// WithdrawResponse представляет ответ с информацией о выводе средств
type WithdrawResponse struct {
	Order       string `json:"order"`
	Sum         uint64 `json:"sum"`
	ProcessedAt string `json:"processed_at"`
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

	// Создаем заказ на списание
	withdrawOrder := *models.MakeWithdraw(*userDB, withdrawReq.Order, withdrawReq.Sum)

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
			Sum:         withdrawal.Value,
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

// func (h *Handler) SetDBString(str string) {
// 	// fmt.Printf("перед присваиванием h.dbConnectionString %s \n", h.dbConnectionString)
// 	// fmt.Printf("перед присваиванием str %s\n", str)
// 	h.dbConnectionString = str
// 	// fmt.Printf("после присваивания h.dbConnectionString %s \n", h.dbConnectionString)
// }

// func (h Handler) UpdateMetric(res http.ResponseWriter, req *http.Request) {
// 	//res.Write([]byte("Привет!"))
// 	//fmt.Println("run update")
// 	if req.Method != http.MethodPost {
// 		// разрешаем только POST-запросы
// 		res.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// 	//TODO проверка на наличие Content-Type: text/plain

// 	//	req.URL
// 	elems := strings.Split(req.URL.Path, "/")

// 	if len(elems) != 5 {
// 		http.Error(res, fmt.Sprintf("неверное количество параметров: %v, все элементы: %v \r\n", len(elems), elems), http.StatusNotFound)
// 		return
// 	}

// 	typeE := elems[2]
// 	nameE := elems[3]
// 	valueE := elems[4]

// 	if !(typeE == "counter" || typeE == "gauge") {
// 		http.Error(res, fmt.Sprintf("Некорректный тип метрики: %v, все элементы: %v \r\n", typeE, elems), http.StatusBadRequest)
// 		return
// 	}

// 	if nameE == "" {
// 		http.Error(res, fmt.Sprintf("Пустое имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
// 		return
// 	}

// 	switch typeE {
// 	case "counter":
// 		val, err := strconv.ParseInt(elems[4], 10, 64)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Некорректное значение метрики: %v, все элементы: %v \r\n", valueE, elems), http.StatusBadRequest)
// 			return
// 		}
// 		h.repo.UpdateCounter(nameE, val)
// 	case "gauge":
// 		val, err := strconv.ParseFloat(elems[4], 64)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Некорректное значение метрики: %v, все элементы: %v \r\n", valueE, elems), http.StatusBadRequest)
// 			return
// 		}
// 		h.repo.UpdateGauge(nameE, val)
// 	}

// 	res.Write([]byte(fmt.Sprintf("elems: %v repo: %v \r\n", elems, h.repo)))
// 	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

// 	fmt.Println(req.URL)
// }

// func (h Handler) GetMetric(res http.ResponseWriter, req *http.Request) {
// 	//res.Write([]byte("Привет!"))
// 	//fmt.Println("run get")
// 	// if req.Method != http.MethodGet {
// 	// 	// разрешаем только POST-запросы
// 	// 	res.WriteHeader(http.StatusMethodNotAllowed)
// 	// 	return
// 	// }

// 	//TODO проверка на наличие Content-Type: text/plain

// 	//	req.URL
// 	elems := strings.Split(req.URL.Path, "/")

// 	if len(elems) != 4 {
// 		http.Error(res, fmt.Sprintf("неверное количество параметров: %v, все элементы: %v \r\n", len(elems), elems), http.StatusNotFound)
// 		return
// 	}

// 	typeE := chi.URLParam(req, "metric_type")
// 	nameE := chi.URLParam(req, "metric_name")
// 	var stringValue string

// 	if !(typeE == "counter" || typeE == "gauge") {
// 		http.Error(res, fmt.Sprintf("Некорректный тип метрики: %v, все элементы: %v \r\n", typeE, elems), http.StatusBadRequest)
// 		return
// 	}

// 	if nameE == "" {
// 		http.Error(res, fmt.Sprintf("Пустое имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
// 		return
// 	}

// 	switch typeE {
// 	case "counter":
// 		v, err := h.repo.GetCounter(nameE)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
// 			return
// 		}

// 		stringValue = strconv.FormatInt(v, 10)
// 	case "gauge":
// 		v, err := h.repo.GetGauge(nameE)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
// 			return
// 		}

// 		stringValue = strconv.FormatFloat(v, 'f', -1, 64)
// 	}

// 	res.Write([]byte(stringValue))
// 	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

// 	fmt.Println(req.URL, stringValue)
// }

// func (h Handler) GetMain(res http.ResponseWriter, req *http.Request) {
// 	const formStart = `<html>
// <head>
// <title>Известные метрики:</title>
//     </head>
//     <body>
// 	`

// 	//<label>Логин <input type="text" name="login"></label>
// 	//<label>Пароль <input type="password" name="password"></label>

// 	const formEnd = `
//     </body>
// </html>`

// 	var formMetrics = `<label>Метрики gauges:</label><br/>`
// 	gaugesKeys := h.repo.GetGaugesKeys()

// 	for _, vkey := range gaugesKeys {
// 		vv, err := h.repo.GetGauge(vkey)
// 		if err == nil {
// 			formMetrics += fmt.Sprintf(`<label>%s = %f</label><br/>`, vkey, vv)
// 		} else {
// 			formMetrics += fmt.Sprintf(`<label>%s = READ ERROR</label><br/>`, vkey)
// 		}
// 	}

// 	formMetrics += `<label>Метрики counters:</label><br/>`
// 	countersKeys := h.repo.GetCountersKeys()

// 	for _, vkey := range countersKeys {
// 		vv, err := h.repo.GetCounter(vkey)
// 		if err == nil {
// 			formMetrics += fmt.Sprintf(`<label>%s = %d</label><br/>`, vkey, vv)
// 		} else {
// 			formMetrics += fmt.Sprintf(`<label>%s = READ ERROR</label><br/>`, vkey)
// 		}
// 	}

// 	var form = formStart + formMetrics + formEnd

// 	res.Header().Set("Content-Type", "text/html ; charset=utf-8")
// 	//res.Header().Set("Content-Type", "")

// 	res.WriteHeader(http.StatusOK)
// 	res.Write([]byte(form))

// 	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

// 	fmt.Println(req.URL)
// }

// func (h Handler) UpdateJSON(res http.ResponseWriter, req *http.Request) {

// 	if req.Method != http.MethodPost {
// 		res.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// 	if req.Header.Get("Content-Type") != "application/json" {
// 		res.WriteHeader(http.StatusResetContent)
// 		return
// 	}

// 	var metric models.Metrics

// 	var buf bytes.Buffer
// 	// читаем тело запроса
// 	_, err := buf.ReadFrom(req.Body)
// 	if err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	// десериализуем JSON в Metric
// 	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	switch metric.MType {
// 	case "counter":
// 		if metric.Delta == nil {
// 			http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
// 			return
// 		}

// 		err := h.repo.UpdateCounter(metric.ID, *metric.Delta)
// 		if err != nil {
// 			http.Error(res, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 	case "gauge":
// 		if metric.Value == nil {
// 			http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
// 			return
// 		}

// 		err := h.repo.UpdateGauge(metric.ID, *metric.Value)
// 		if err != nil {
// 			http.Error(res, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 	default:
// 		http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusBadRequest)
// 		return
// 	}

// 	res.WriteHeader(http.StatusOK)
// }

// func (h Handler) UpdatesJSON(res http.ResponseWriter, req *http.Request) {

// 	fmt.Println("===handlers start updates")
// 	defer fmt.Println("===handlers finish updates")

// 	if req.Method != http.MethodPost {
// 		res.WriteHeader(http.StatusMethodNotAllowed)
// 		fmt.Println("-=UpdatesJSON:   err http.MethodPost")
// 		return
// 	}

// 	if req.Header.Get("Content-Type") != "application/json" {
// 		res.WriteHeader(http.StatusResetContent)
// 		fmt.Println("-=UpdatesJSON:   err req.Header.Get Content-Type...")
// 		return
// 	}

// 	//var metric models.Metrics

// 	var metrics []models.Metrics

// 	var buf bytes.Buffer
// 	// читаем тело запроса
// 	_, err := buf.ReadFrom(req.Body)
// 	if err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		fmt.Println("-=UpdatesJSON:   err ReadFrom(req.Body)")
// 		return
// 	}
// 	// десериализуем JSON в Metric
// 	if err = json.Unmarshal(buf.Bytes(), &metrics); err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		fmt.Println("-=UpdatesJSON:   err json.Unmarshal")
// 		return
// 	}

// 	if massUpdater, ok := h.repo.(repository.MassUpdater); ok {
// 		err := massUpdater.MassUpdate(metrics)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("mass updater выдал ошибку: %v, err = %s \r\n", metrics, err), http.StatusInternalServerError)
// 			return
// 		}
// 	} else {
// 		for _, metric := range metrics {
// 			switch metric.MType {
// 			case "counter":
// 				if metric.Delta == nil {
// 					http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
// 					return
// 				}

// 				err := h.repo.UpdateCounter(metric.ID, *metric.Delta)
// 				if err != nil {
// 					http.Error(res, err.Error(), http.StatusInternalServerError)
// 					return
// 				}

// 			case "gauge":
// 				if metric.Value == nil {
// 					http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
// 					return
// 				}

// 				err := h.repo.UpdateGauge(metric.ID, *metric.Value)
// 				if err != nil {
// 					http.Error(res, err.Error(), http.StatusInternalServerError)
// 					return
// 				}
// 			default:
// 				http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusBadRequest)
// 				return
// 			}
// 		}
// 	}

// 	fmt.Println("   before status ok")
// 	res.WriteHeader(http.StatusOK)
// 	fmt.Println("   after status ok")
// }

// func (h Handler) RegisterUser(res http.ResponseWriter, req *http.Request) {
// 	//Content-Type application/json
// 	if req.Method != http.MethodPost {
// 		res.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// 	if req.Header.Get("Content-Type") != "application/json" {
// 		res.WriteHeader(http.StatusResetContent)
// 		return
// 	}

// 	var metric models.Metrics
// 	var metricOut models.Metrics
// 	var buf bytes.Buffer
// 	// читаем тело запроса
// 	_, err := buf.ReadFrom(req.Body)
// 	if err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	// десериализуем JSON в Metric
// 	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
// 		http.Error(res, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	switch metric.MType {
// 	case "counter":
// 		v, err := h.repo.GetCounter(metric.ID)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v \r\n", metric.ID), http.StatusNotFound)
// 			return
// 		}

// 		metricOut.Delta = &v
// 	case "gauge":
// 		v, err := h.repo.GetGauge(metric.ID)
// 		if err != nil {
// 			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v \r\n", metric.ID), http.StatusNotFound)
// 			return
// 		}

// 		metricOut.Value = &v
// 	default:
// 		http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusNotFound)
// 		return
// 	}

// 	metricOut.MType = metric.MType
// 	metricOut.ID = metric.ID

// 	resp, err := json.Marshal(metricOut)
// 	if err != nil {
// 		http.Error(res, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	res.Header().Set("Content-Type", "application/json")
// 	res.WriteHeader(http.StatusOK)
// 	res.Write(resp)
// }

// func (h Handler) PingDB(res http.ResponseWriter, req *http.Request) {

// 	if pinger, ok := h.repo.(repository.Pinger); ok {
// 		if err := pinger.Ping(); err != nil {
// 			http.Error(res, fmt.Sprintf("Ошибка: %v \r\n", err), http.StatusInternalServerError)
// 			return
// 		}
// 	}

// 	res.WriteHeader(http.StatusOK)

// }
