package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"api-service/internal/kafka"
	"api-service/internal/model"
	"api-service/internal/repository"
	"github.com/google/uuid"
)

type AuthService struct {
	users       *repository.UserRepository
	producer    kafka.Producer
	topic       string
	logger      *slog.Logger
}

func NewAuthService(users *repository.UserRepository, producer kafka.Producer, topic string, logger *slog.Logger) *AuthService {
	return &AuthService{
		users:    users,
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

func (s *AuthService) Register(ctx context.Context, username string) (model.User, error) {
	user := model.User{
		ID:       uuid.New().String(),
		Username: username,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return model.User{}, err
	}

	event := map[string]string{
		"user_id":   user.ID,
		"username":  user.Username,
		"event_type": "user.registered",
	}
	if err := s.producer.Publish(ctx, s.topic, user.ID, event); err != nil {
		s.logger.Error("kafka publish failed", "component", "service.auth", "operation", "register.publish", "user_id", user.ID, "error", err)
		return model.User{}, fmt.Errorf("publish user.registered: %w", err)
	}

	return user, nil
}

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
