package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"api-service/internal/model"
	"api-service/internal/repository"

	"github.com/google/uuid"
)

// AuthService отвечает за регистрацию и вход пользователей.
type AuthService struct {
	users  *repository.UserRepository
	logger *slog.Logger
}

// NewAuthService создает сервис аутентификации.
func NewAuthService(users *repository.UserRepository, logger *slog.Logger) *AuthService {
	return &AuthService{
		users:  users,
		logger: logger,
	}
}

// Register создает пользователя с новым UUID.
func (s *AuthService) Register(ctx context.Context, username string) (model.User, error) {
	user := model.User{
		ID:       uuid.New().String(),
		Username: username,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return model.User{}, err
	}

	return user, nil
}

// Login возвращает пользователя по username или ошибку "не найден".
func (s *AuthService) Login(ctx context.Context, username string) (model.User, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}
	return user, nil
}
