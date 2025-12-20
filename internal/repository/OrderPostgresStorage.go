package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type OrderPostgresStorage struct {
	db *PostgresConnection
}

func MakeOrderPostgresStorage(pc *PostgresConnection) *OrderPostgresStorage {

	return &OrderPostgresStorage{
		db: pc,
	}
}

func (st *OrderPostgresStorage) AddOrder(user models.User, order models.Order) error {
	// Проверяем корректность номера заказа по алгоритму Луна
	if !models.LunaCheck(order.OrderID) {
		return ErrBadOrderID
	}

	// Получаем ID пользователя
	var userID uint64
	query := "SELECT id FROM gophermart_users WHERE login = $1"
	err := st.db.db.QueryRow(query, user.Login).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBadLogin
		}
		return fmt.Errorf("ошибка при получении ID пользователя: %w", err)
	}

	// Проверяем, существует ли уже такой заказ у этого пользователя
	var existingUserID uint64
	checkQuery := "SELECT user_id FROM gophermart_orders WHERE id = $1"
	err = st.db.db.QueryRow(checkQuery, order.OrderID).Scan(&existingUserID)
	if err == nil {
		// Заказ существует, проверяем у какого пользователя
		if existingUserID == userID {
			return ErrOrderExistThisUser
		}
		return ErrOrderExistAnotherUser
	} else if !errors.Is(err, sql.ErrNoRows) {
		// Произошла другая ошибка при проверке
		return fmt.Errorf("ошибка при проверке существования заказа: %w", err)
	}

	// Проверяем тип заказа
	if order.Type != models.OrderType && order.Type != models.WithdrawType {
		return ErrOrderType
	}

	// Преобразуем строку даты в time.Time
	createdAt, err := time.Parse(time.RFC3339, order.Date)
	if err != nil {
		// Если не удалось распарсить дату, используем текущее время
		createdAt = time.Now()
	}

	// Если это операция списания, используем транзакцию для атомарности
	if order.Type == models.WithdrawType {
		// Начинаем транзакцию
		tx, err := st.db.db.Begin()
		if err != nil {
			return fmt.Errorf("ошибка при начале транзакции: %w", err)
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()

		// Проверяем текущий баланс пользователя в рамках транзакции
		var currentBalance uint64
		balanceQuery := `
			SELECT COALESCE(SUM(CASE WHEN type = 'ORDER' THEN value ELSE 0 END), 0) -
			       COALESCE(SUM(CASE WHEN type = 'WITHDRAW' THEN value ELSE 0 END), 0)
			FROM gophermart_orders
			WHERE user_id = $1
		`
		err = tx.QueryRow(balanceQuery, userID).Scan(&currentBalance)
		if err != nil {
			return fmt.Errorf("ошибка при получении баланса в транзакции: %w", err)
		}

		// Проверяем, достаточно ли средств для списания
		if order.Value > currentBalance {
			return ErrIncafitionFunds
		}

		// Добавляем заказ на списание в рамках транзакции
		insertQuery := `
			INSERT INTO gophermart_orders (id, user_id, type, status, value, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = tx.Exec(insertQuery, order.OrderID, userID, order.Type, order.Status, order.Value, createdAt)
		if err != nil {
			// Проверяем ошибку уникального ограничения на случай гонки состояний
			if isUniqueViolationError(err) {
				// Повторно проверяем, у какого пользователя существует заказ
				err = tx.QueryRow(checkQuery, order.OrderID).Scan(&existingUserID)
				if err == nil {
					if existingUserID == userID {
						return ErrOrderExistThisUser
					}
					return ErrOrderExistAnotherUser
				}
			}
			return fmt.Errorf("ошибка при добавлении заказа в транзакции: %w", err)
		}

		// Подтверждаем транзакцию
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("ошибка при подтверждении транзакции: %w", err)
		}
	} else {
		// Для обычных заказов (не списаний) добавляем без транзакции
		insertQuery := `
			INSERT INTO gophermart_orders (id, user_id, type, status, value, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = st.db.db.Exec(insertQuery, order.OrderID, userID, order.Type, order.Status, order.Value, createdAt)
		if err != nil {
			// Проверяем ошибку уникального ограничения на случай гонки состояний
			if isUniqueViolationError(err) {
				// Повторно проверяем, у какого пользователя существует заказ
				err = st.db.db.QueryRow(checkQuery, order.OrderID).Scan(&existingUserID)
				if err == nil {
					if existingUserID == userID {
						return ErrOrderExistThisUser
					}
					return ErrOrderExistAnotherUser
				}
			}
			return fmt.Errorf("ошибка при добавлении заказа: %w", err)
		}
	}

	return nil
}

func (st *OrderPostgresStorage) GetOrders(user models.User, orderType string) ([]models.Order, error) {
	// Получаем ID пользователя
	var userID uint64
	query := "SELECT id FROM gophermart_users WHERE login = $1"
	err := st.db.db.QueryRow(query, user.Login).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.Order{}, nil
		}
		return nil, fmt.Errorf("ошибка при получении ID пользователя: %w", err)
	}

	// Формируем запрос в зависимости от типа заказа
	var ordersQuery string
	var args []interface{}

	if orderType == "" {
		// Если тип не указан, получаем все заказы
		ordersQuery = `
			SELECT id, type, status, value, created_at
			FROM gophermart_orders
			WHERE user_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{userID}
	} else {
		// Если тип указан, фильтруем по типу
		ordersQuery = `
			SELECT id, type, status, value, created_at
			FROM gophermart_orders
			WHERE user_id = $1 AND type = $2
			ORDER BY created_at DESC
		`
		args = []interface{}{userID, orderType}
	}

	rows, err := st.db.db.Query(ordersQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении заказов: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var createdAt time.Time

		err := rows.Scan(&order.OrderID, &order.Type, &order.Status, &order.Value, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании заказа: %w", err)
		}

		// Устанавливаем пользователя и дату
		order.User = user.Login
		order.Date = createdAt.Format(time.RFC3339)

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по заказам: %w", err)
	}

	return orders, nil
}

func (st *OrderPostgresStorage) GetBalance(user models.User) (*models.Balance, error) {
	// Получаем ID пользователя
	var userID uint64
	query := "SELECT id FROM gophermart_users WHERE login = $1"
	err := st.db.db.QueryRow(query, user.Login).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Balance{Current: 0, Withdrawn: 0}, nil
		}
		return nil, fmt.Errorf("ошибка при получении ID пользователя: %w", err)
	}

	// Получаем сумму всех заказов и списаний
	balanceQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'ORDER' THEN value ELSE 0 END), 0) as current,
			COALESCE(SUM(CASE WHEN type = 'WITHDRAW' THEN value ELSE 0 END), 0) as withdrawn
		FROM gophermart_orders
		WHERE user_id = $1
	`

	var balance models.Balance
	err = st.db.db.QueryRow(balanceQuery, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении баланса: %w", err)
	}

	return &balance, nil
}

// GetOrdersWithStatuses получает заказы с указанными статусами
func (st *OrderPostgresStorage) GetOrdersWithStatuses(statuses []string) ([]models.Order, error) {
	if len(statuses) == 0 {
		return []models.Order{}, nil
	}

	// Формируем запрос с использованием IN оператора
	query := `
		SELECT id, user_id, type, status, value, created_at
		FROM gophermart_orders
		WHERE status = ANY($1)
		ORDER BY created_at ASC
	`

	rows, err := st.db.db.Query(query, statuses)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении заказов по статусам: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var userID uint64
		var createdAt time.Time

		err := rows.Scan(&order.OrderID, &userID, &order.Type, &order.Status, &order.Value, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании заказа: %w", err)
		}

		// Получаем логин пользователя по ID
		var userLogin string
		userQuery := "SELECT login FROM gophermart_users WHERE id = $1"
		err = st.db.db.QueryRow(userQuery, userID).Scan(&userLogin)
		if err != nil {
			return nil, fmt.Errorf("ошибка при получении логина пользователя: %w", err)
		}

		// Устанавливаем пользователя и дату
		order.User = userLogin
		order.Date = createdAt.Format(time.RFC3339)

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по заказам: %w", err)
	}

	return orders, nil
}

// UpdateOrderStatusAndValue обновляет статус и значение заказа
func (st *OrderPostgresStorage) UpdateOrderStatusAndValue(orderID, status string, value uint64) error {
	query := `
		UPDATE gophermart_orders
		SET status = $1, value = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := st.db.db.Exec(query, status, value, orderID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса заказа: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при получении количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("заказ с ID %s не найден", orderID)
	}

	return nil
}

// GetWithdrawals получает историю выводов средств пользователя
func (st *OrderPostgresStorage) GetWithdrawals(user models.User) ([]models.Order, error) {
	// Получаем ID пользователя
	var userID uint64
	query := "SELECT id FROM gophermart_users WHERE login = $1"
	err := st.db.db.QueryRow(query, user.Login).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.Order{}, nil
		}
		return nil, fmt.Errorf("ошибка при получении ID пользователя: %w", err)
	}

	// Получаем только операции списания
	withdrawalsQuery := `
		SELECT id, type, status, value, created_at
		FROM gophermart_orders
		WHERE user_id = $1 AND type = $2
		ORDER BY created_at DESC
	`

	rows, err := st.db.db.Query(withdrawalsQuery, userID, models.WithdrawType)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении истории выводов: %w", err)
	}
	defer rows.Close()

	var withdrawals []models.Order
	for rows.Next() {
		var withdrawal models.Order
		var createdAt time.Time

		err := rows.Scan(&withdrawal.OrderID, &withdrawal.Type, &withdrawal.Status, &withdrawal.Value, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании вывода: %w", err)
		}

		// Устанавливаем пользователя и дату
		withdrawal.User = user.Login
		withdrawal.Date = createdAt.Format(time.RFC3339)

		withdrawals = append(withdrawals, withdrawal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по выводам: %w", err)
	}

	return withdrawals, nil
}
