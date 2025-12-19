package models

import "time"

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"

	OrderType    = "ORDER"
	WithdrawType = "WITHDRAW"
)

type money uint64

type User struct {
	UserID   *uint64 `json:"user_id,omitempty"`
	Login    string  `json:"login"`
	Password string  `json:"password"`
}

type Order struct {
	OrderID uint64 `json:"number"`
	User    string `json:"-"`
	Type    string `json:"-"`
	Status  string `json:"status"`
	Date    string `json:"uploaded_at"` //TODO может быть переделать на дату всё же?
	Value   uint64 `json:"-"`
}

// type Order struct {
// 	OrderID uint64 `json:"order_id,omitempty"`
// 	User    string `json:"user"`
// 	Type    string `json:"type"`
// 	Status  string `json:"status"`
// 	Date    string `json:"date"` //TODO может быть переделать на дату всё же?
// 	Value   uint64 `json:"value"`
// }

type Balance struct {
	Current   uint64 `json:"current"`
	Withdrawn uint64 `json:"withdrawn"`
}

func MakeNewOrder(user User, orderID uint64) *Order {

	return &Order{
		OrderID: orderID,
		User:    user.Login,
		Type:    OrderType,
		Status:  OrderStatusNew,
		Date:    time.Now().Format(time.RFC3339),
		Value:   0,
	}
}

func MakeWithdraw(user User, orderID uint64, sum uint64) *Order {

	return &Order{
		OrderID: orderID,
		User:    user.Login,
		Type:    OrderType,
		Status:  OrderStatusNew,
		Date:    time.Now().Format(time.RFC3339),
		Value:   sum,
	}
}
