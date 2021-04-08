package store

import "github.com/zlyaptica/http-rest-api/internal/app/model"

// UserRepository ...
type UserRepository interface {
	Create(*model.User) error
	Find(int) (*model.User, error)
	FindByEmail(string) (*model.User, error)
	FindByID(int) (*model.User, error)
}

// PostRepository ...
type PostRepository interface {
	Create(*model.Post) error
	Delete(int) error
	Update(string, string, int) error
	FindByAuthor(int) ([]model.Post, error)
	Find(int) (*model.Post, error)
	FindAll() ([]model.Post, error)
	FindN(int, int) ([]model.Post, error)
	IsStarredByUser(int, int) (bool, error)
	GetStarsCount(int) (int, error)
}

type StarRepository interface {
	Create(*model.Star) error
	Delete(int, int) error
}
