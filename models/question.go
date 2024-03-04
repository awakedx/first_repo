package models

type Question struct {
	Question string `json:"question"`
	Answer   string `json:"string"`
	UserId   int64  `json:"userid"`
	Id       int64  `json:"id"`
}
