package repository

import (
	"sync"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type OrderMemStorage struct {
	orders map[string][]models.Order
	mutex  sync.Mutex //TODO добавить мутекс в каждого пользователя и блокировать попользовательно
}

func MakeOrderMemStorage() *OrderMemStorage {

	return &OrderMemStorage{
		orders: make(map[string][]models.Order),
	}
}

func (st *OrderMemStorage) AddOrder(user models.User, order models.Order) error {

	st.mutex.Lock()
	defer st.mutex.Unlock()

	orders, ok := st.orders[user.Login]

	if !ok {
		orders = make([]models.Order, 0, 10)
	}

	for _, v := range orders {
		//fmt.Printf("z1 %s %s %v \n", v.OrderID, order.OrderID, v.OrderID == order.OrderID)
		if v.OrderID == order.OrderID {
			return ErrOrderExistThisUser
		}
	}

	for _, list := range st.orders {
		for _, v := range list {
			//fmt.Printf("z2 %s %s %v \n", v.OrderID, order.OrderID, v.OrderID == order.OrderID)
			if v.OrderID == order.OrderID {
				return ErrOrderExistAnotherUser
			}
		}
	}

	if order.Type == models.WithdrawType {

		var sumOrder uint64
		for _, v := range orders {
			if v.Type == models.OrderType {
				sumOrder += v.Value
			}
		}

		if order.Value > sumOrder {
			return ErrIncafitionFunds
		}

	} else if order.Type != models.OrderType {
		return ErrOrderType
	}

	orders = append(orders, order)

	st.orders[user.Login] = orders
	return nil

}

func (st *OrderMemStorage) GetOrders(user models.User, orderType string) ([]models.Order, error) {
	orders, ok := st.orders[user.Login]
	if !ok {
		return make([]models.Order, 0), nil
	}

	ordersTyped := make([]models.Order, 0, 10)

	for _, v := range orders {
		if v.Type == orderType {
			ordersTyped = append(ordersTyped, v)
		}
	}

	return ordersTyped, nil
}

func (st *OrderMemStorage) GetBalance(user models.User) (*models.Balance, error) {

	orders, ok := st.orders[user.Login]
	if !ok {
		return &models.Balance{
				Current:   0,
				Withdrawn: 0,
			},
			nil
	}

	var sumOrder, sumWithdraw uint64

	for _, v := range orders {
		switch v.Type {
		case models.OrderType:
			sumOrder += v.Value
		case models.WithdrawType:
			sumWithdraw += v.Value
		}
		//TODO что делать если обнаружили невалидный тип??

	}

	return &models.Balance{
			Current:   sumOrder,
			Withdrawn: sumWithdraw,
		},
		nil
}
