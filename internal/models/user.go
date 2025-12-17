package models

type User struct {
	ID       string `json:"id,omitempty"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
