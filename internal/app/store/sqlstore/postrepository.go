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
		p.Author.ID,
		p.Header,
		p.TextPost,
		time.Now(),
	).Scan(&p.ID)
}

// Delete ...
func (r *PostRepository) Delete(id int) error {
	_, err := r.store.db.Query("DELETE FROM stars WHERE post_id = $1", id)
	_, err = r.store.db.Query("DELETE FROM posts WHERE id = $1", id)
	return err
}

// Update ...
func (r *PostRepository) Update(header string, text_post string, id int) error {
	_, err := r.store.db.Query("UPDATE posts SET (header, text_post) = ($1, $2) WHERE id = $3", header, text_post, id)
	return err
}

// IsStarredByUser ...
func (r *PostRepository) IsStarredByUser(userID int, postID int) (bool, error) {
	var count int
	if err := r.store.db.QueryRow(
		"SELECT COUNT(*) FROM stars WHERE liker_id = $1 AND post_id = $2",
		userID,
		postID,
	).Scan(
		&count,
	); err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetStartCount ...
func (r *PostRepository) GetStarsCount(postID int) (int, error) {
	var count int
	if err := r.store.db.QueryRow(
		"SELECT COUNT(*) FROM stars WHERE post_id = $1",
		postID,
	).Scan(
		&count,
	); err != nil {
		return 0, err
	}

	return count, nil
}

// Find ...
func (r *PostRepository) Find(id int) (*model.Post, error) {
	u := &model.User{}
	p := &model.Post{
		Author: u,
	}
	if err := r.store.db.QueryRow(
		"SELECT users.username, users.id, posts.id, posts.header, posts.text_post, posts.created_at FROM posts INNER JOIN users ON posts.author_id = users.id WHERE posts.id = $1",
		id,
	).Scan(
		&p.Author.Username,
		&p.Author.ID,
		&p.ID,
		&p.Header,
		&p.TextPost,
		&p.CreatedAt,
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
	rows, err := r.store.db.Query(
		"SELECT users.username, users.id, posts.id, posts.header, posts.text_post, posts.created_at FROM posts INNER JOIN users ON posts.author_id = users.id ORDER BY posts.id DESC",
	)
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}

	for rows.Next() {
		u := &model.User{}
		p := model.Post{
			Author: u,
		}
		err := rows.Scan(
			&p.Author.Username,
			&p.Author.ID,
			&p.ID,
			&p.Header,
			&p.TextPost,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		p.StarsCount, err = r.GetStarsCount(p.ID)

		posts = append(posts, p)
	}

	return posts, nil
}

// FindByAuthor ...
func (r *PostRepository) FindByAuthor(id int) ([]model.Post, error) {
	posts := []model.Post{}
	rows, err := r.store.db.Query(
		"SELECT users.username, users.id, posts.id, posts.header, posts.text_post, posts.created_at FROM posts INNER JOIN users ON posts.author_id = users.id WHERE users.id = $1 ORDER BY posts.id DESC",
		id,
	)
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}

	for rows.Next() {
		u := &model.User{}
		p := model.Post{
			Author: u,
		}
		err := rows.Scan(
			&p.Author.Username,
			&p.Author.ID,
			&p.ID,
			&p.Header,
			&p.TextPost,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		p.StarsCount, err = r.GetStarsCount(p.ID)
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
		p.Author,
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
