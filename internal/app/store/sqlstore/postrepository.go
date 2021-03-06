package sqlstore

import (
	"database/sql"
	"time"

	"github.com/zlyaptica/http-rest-api/internal/app/model"
	"github.com/zlyaptica/http-rest-api/internal/app/store"
)

// PostRepository ...
type PostRepository struct {
	store *Store
}

// Create ...
func (r *PostRepository) Create(p *model.Post) error {
	if err := p.Validate(); err != nil {
		return err
	}

	return r.store.db.QueryRow(
		"INSERT INTO posts (author_id, header, text_post, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
		p.AuthorID,
		p.Header,
		p.TextPost,
		time.Now(),
	).Scan(&p.ID)
}

// Find ...
func (r *PostRepository) Find(id int) (*model.Post, error) {
	p := &model.Post{}
	if err := r.store.db.QueryRow(
		"SELECT id, author_id, header, text_post, created_at FROM posts WHERE id = $1",
		id,
	).Scan(
		p.ID,
		p.AuthorID,
		p.Header,
		p.TextPost,
		p.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}

		return nil, err
	}

	return p, nil
}

// FindAll ...
func (r *PostRepository) FindAll() ([]model.Post, error) {
	posts := []model.Post{}
	rows, err := r.store.db.Query("SELECT id, author_id, header, text_post, created_at FROM posts")
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}

	for rows.Next() {
		p := model.Post{}
		err := rows.Scan(&p.ID, &p.AuthorID, &p.Header, &p.TextPost, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}

// FindN ...
func (r *PostRepository) FindN(id int, n int) ([]model.Post, error) {
	p := model.Post{}
	if err := r.store.db.QueryRow(
		"SELECT id, author_id, header, text_post, created_at FROM posts WHERE id = $1",
		id,
	).Scan(
		p.ID,
		p.AuthorID,
		p.Header,
		p.TextPost,
		p.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}

		return nil, err
	}

	return []model.Post{p}, nil
}
