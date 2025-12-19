package models

type User struct {
	ID       *uint64 `json:"id,omitempty"`
	Login    string  `json:"login"`
	Password string  `json:"password"`
}
