package sqlstore

import (
	"github.com/jmoiron/sqlx"
	"github.com/zlyaptica/http-rest-api/internal/app/store"
)

// Store ...
type Store struct {
	db             *sqlx.DB
	userRepository *UserRepository
	postRepository *PostRepository
	starRepository *StarRepository
}

// New ...
func New(db *sqlx.DB) *Store {
	return &Store{
		db: db,
	}
}

// User ...
func (s *Store) User() store.UserRepository {
	if s.userRepository != nil {
		return s.userRepository
	}

	s.userRepository = &UserRepository{
		store: s,
	}

	return s.userRepository
}

// Post ...
func (s *Store) Post() store.PostRepository {
	if s.postRepository != nil {
		return s.postRepository
	}

	s.postRepository = &PostRepository{
		store: s,
	}

	return s.postRepository
}

// Star ...
func (s *Store) Star() store.StarRepository {
	if s.starRepository != nil {
		return s.starRepository
	}

	s.starRepository = &StarRepository{
		store: s,
	}

	return s.starRepository
}
