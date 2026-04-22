package service

import (
	"context"

	"api-service/internal/model"
	"api-service/internal/repository"
)

type MessageService struct {
	messages *repository.MessageRepository
}

func NewMessageService(messages *repository.MessageRepository) *MessageService {
	return &MessageService{messages: messages}
}

func (s *MessageService) ListByChatID(ctx context.Context, chatID string, limit string) ([]model.Message, error) {
	return s.messages.ListByChatID(ctx, chatID, limit)
}
