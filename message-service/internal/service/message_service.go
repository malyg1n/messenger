package service

import (
	"context"
	"encoding/json"
	"fmt"

	"message-service/internal/model"
)

type messageRepository interface {
	Save(ctx context.Context, message model.ChatMessage) error
}

type messagePublisher interface {
	Publish(ctx context.Context, key []byte, value []byte) error
}

type MessageService struct {
	repo      messageRepository
	publisher messagePublisher
}

// NewMessageService создает сервис обработки входящих сообщений.
func NewMessageService(repo messageRepository, publisher messagePublisher) *MessageService {
	return &MessageService{
		repo:      repo,
		publisher: publisher,
	}
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

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode saved message: %w", err)
	}
	if err := s.publisher.Publish(ctx, []byte(message.ChatID), payload); err != nil {
		return fmt.Errorf("publish saved message: %w", err)
	}

	return nil
}
