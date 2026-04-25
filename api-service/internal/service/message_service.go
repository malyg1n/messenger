package service

import (
	"context"

	"api-service/internal/model"
	"api-service/internal/repository"
)

// MessageService предоставляет use-case чтения сообщений чата.
type MessageService struct {
	messages *repository.MessageRepository
}

// NewMessageService создает сервис сообщений.
func NewMessageService(messages *repository.MessageRepository) *MessageService {
	return &MessageService{messages: messages}
}

// ListByChatID возвращает историю сообщений с пагинацией по курсору before.
func (s *MessageService) ListByChatID(ctx context.Context, chatID string, before string, limit string) ([]model.Message, error) {
	return s.messages.ListByChatID(ctx, chatID, before, limit)
}
