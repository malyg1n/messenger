package service

import (
	"context"

	"api-service/internal/model"
	"api-service/internal/repository"
)

type UserService struct {
	users *repository.UserRepository
}

func NewUserService(users *repository.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.users.List(ctx)
}
