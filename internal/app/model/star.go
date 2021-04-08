package model

type Star struct {
	ID     int `json:"id"`
	Starer *User 
	Post   *Post `json:"post"`
}
