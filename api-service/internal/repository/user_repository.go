package repository

import (
	"context"
	"database/sql"

	"api-service/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user model.User) error {
	_, err := r.db.ExecContext(ctx, "insert into users(id, username) values ($1,$2)", user.ID, user.Username)
	return err
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx, "select id, username from users order by created_at desc")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(
		ctx,
		`select id, username from users where username = $1`,
		username,
	).Scan(&user.ID, &user.Username)
	return user, err
}
