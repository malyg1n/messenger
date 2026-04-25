package service

import (
	"context"

	"api-service/internal/model"
	"api-service/internal/repository"
)

// UserService предоставляет операции чтения пользователей для HTTP-слоя.
type UserService struct {
	users *repository.UserRepository
}

// NewUserService создает сервис пользователей.
func NewUserService(users *repository.UserRepository) *UserService {
	return &UserService{users: users}
}

// List возвращает список пользователей.
func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.users.List(ctx)
}
