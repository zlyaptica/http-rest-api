package model

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
)

// Post ...
type Post struct {
	ID        int       `json:"id"`
	Author    *User     `json:"author"`
	Header    string    `json:"header"`
	TextPost  string    `json:"text_post"`
	CreatedAt time.Time `json:"created_at"`
}

// Validate ...
func (p *Post) Validate() error {
	return validation.ValidateStruct(
		p,
		validation.Field(&p.Header, validation.Required, validation.Length(8, 128)),
		validation.Field(&p.TextPost, validation.Required, validation.Length(5, 10000)),
	)
}
