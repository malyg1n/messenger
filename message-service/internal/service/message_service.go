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

func NewMessageService(repo messageRepository) *MessageService {
	return &MessageService{repo: repo}
}

func (s *MessageService) Process(ctx context.Context, message model.ChatMessage) error {
	if message.ChatID == "" || message.SenderID == "" || message.Body == "" {
		return fmt.Errorf("message contains empty required fields")
	}

	if err := s.repo.Save(ctx, message); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	return nil
}
