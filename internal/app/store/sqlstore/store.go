package sqlstore

import (
	"database/sql"

	"github.com/zlyaptica/http-rest-api/internal/app/store"
)

// Store ...
type Store struct {
	db             *sql.DB
	userRepository *UserRepository
	postRepository *PostRepository
}

// New ...
func New(db *sql.DB) *Store {
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
