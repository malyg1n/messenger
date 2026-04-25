package service

import (
	"context"
	"fmt"

	"message-service/internal/model"
)

type messageRepository interface {
	Save(ctx context.Context, message model.ChatMessage) error
}

type MessageService struct {
	repo messageRepository
}

// NewMessageService создает сервис обработки входящих сообщений.
func NewMessageService(repo messageRepository) *MessageService {
	return &MessageService{repo: repo}
}

// Process валидирует и сохраняет сообщение, полученное из Kafka.
func (s *MessageService) Process(ctx context.Context, message model.ChatMessage) error {
	// Validate централизует все доменные проверки модели.
	if err := message.Validate(); err != nil {
		return fmt.Errorf("validate message: %w", err)
	}

	// После всех проверок сохраняем сообщение в постоянное хранилище.
	if err := s.repo.Save(ctx, message); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	return nil
}
