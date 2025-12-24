package handler

// BalanceExport представляет структуру для экспорта баланса
type BalanceExport struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// OrderExport представляет структуру для экспорта заказов
type OrderExport struct {
	OrderID string   `json:"number"`
	User    string   `json:"-"`
	Type    string   `json:"-"`
	Status  string   `json:"status"`
	Date    string   `json:"uploaded_at"`
	Value   *float64 `json:"accrual,omitempty"`
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
