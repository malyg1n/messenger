package repository

import (
	"context"
	"database/sql"
	"fmt"

	"api-service/internal/model"
)

// UserRepository инкапсулирует доступ к данным пользователей в Postgres.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository создает репозиторий пользователей.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create сохраняет нового пользователя.
func (r *UserRepository) Create(ctx context.Context, user model.User) error {
	_, err := r.db.ExecContext(ctx, "insert into users(id, username) values ($1,$2)", user.ID, user.Username)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// List возвращает пользователей, отсортированных по дате создания.
func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx, "select id, username from users order by created_at desc")
	if err != nil {
		return nil, fmt.Errorf("list users query: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			return nil, fmt.Errorf("list users scan: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users rows: %w", err)
	}

	return users, nil
}

// FindByUsername ищет пользователя по уникальному username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(
		ctx,
		`select id, username from users where username = $1`,
		username,
	).Scan(&user.ID, &user.Username)
	if err != nil {
		return user, fmt.Errorf("find user by username: %w", err)
	}
	return user, nil
}
