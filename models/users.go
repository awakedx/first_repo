package models

type User struct {
	UserId         int64    `json:"user_id"`
	Is_passing     bool     `json:"is_passing"`
	Is_passed      bool     `json:"is_passed"`
	Question_index int64    `json:"quesiton_index"`
	Answers        []string `json:"answers"`
}
