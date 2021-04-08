package sqlstore

import (
	"database/sql"

	"github.com/zlyaptica/http-rest-api/internal/app/model"
	"github.com/zlyaptica/http-rest-api/internal/app/store"
)

type StarRepository struct {
	store *Store
}

// Create ...
func (r *StarRepository) Create(s *model.Star) error {
	return r.store.db.QueryRow(
		"INSERT INTO stars (liker_id, post_id) values ($1, $2) RETURNING id",
		s.Starer.ID,
		s.Post.ID,
	).Scan(&s.ID)
}

// Delete ...
func (r *StarRepository) Delete(userID int, postID int) error {
	_, err := r.store.db.Query(
		"DELETE FROM stars WHERE liker_id = $1 and post_id = $2",
		userID,
		postID,
	)
	return err
}

// Find ...
func (r *PostRepository) FindByPostID(postID int) ([]model.Star, error) {
	stars := []model.Star{}
	rows, err := r.store.db.Query("SELECT id, liker_id, post_id FROM start WHERE id = $1", postID)
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}

	for rows.Next() {
		u := &model.User{}
		p := &model.Post{}
		s := model.Star{
			Starer: u,
			Post:   p,
		}

		err := rows.Scan(
			&s.ID,
			&s.Starer.ID,
			&s.Post.ID,
		)
		if err != nil {
			return nil, err
		}
		stars = append(stars, s)
	}
	return stars, nil
}
