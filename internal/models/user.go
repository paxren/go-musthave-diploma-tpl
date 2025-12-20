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
	OrderID string `json:"number"`
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

func MakeNewOrder(user User, orderID string) *Order {

	return &Order{
		OrderID: orderID,
		User:    user.Login,
		Type:    OrderType,
		Status:  OrderStatusNew,
		Date:    time.Now().Format(time.RFC3339),
		Value:   0,
	}
}

func MakeWithdraw(user User, orderID string, sum uint64) *Order {

	return &Order{
		OrderID: orderID,
		User:    user.Login,
		Type:    WithdrawType,
		Status:  OrderStatusNew,
		Date:    time.Now().Format(time.RFC3339),
		Value:   sum,
	}
}

func LunaCheck(id string) bool {

	// Преобразуем строку в срез цифр
	var digits []int
	for _, char := range id {
		if char >= '0' && char <= '9' {
			digits = append(digits, int(char-'0'))
		} else {
			// Если в строке есть нецифровые символы, номер недействителен
			return false
		}
	}

	// Если номер слишком короткий, он недействителен
	if len(digits) < 2 {
		return false
	}

	// Алгоритм Луна:
	// 1. Начиная справа, удваиваем каждую вторую цифру
	// 2. Если результат удвоения больше 9, вычитаем 9
	// 3. Суммируем все цифры
	// 4. Если сумма делится на 10 без остатка, номер действителен

	sum := 0
	// Определяем, нужно ли удваивать текущую цифру (начиная справа)
	double := false

	// Проходим по цифрам справа налево
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	// Номер действителен, если сумма делится на 10 без остатка
	return sum%10 == 0

}
