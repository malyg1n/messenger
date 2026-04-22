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

type ChatService struct {
	chats     *repository.ChatRepository
	producer  kafka.Producer
	topic     string
	logger    *slog.Logger
}

func NewChatService(chats *repository.ChatRepository, producer kafka.Producer, topic string, logger *slog.Logger) *ChatService {
	return &ChatService{
		chats:    chats,
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

func (s *ChatService) GetOrCreateDirect(ctx context.Context, userID string, targetUserID string) (string, error) {
	chatID, err := s.chats.FindDirectChatID(ctx, userID, targetUserID)
	if err == nil {
		return chatID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	chatID = uuid.New().String()
	if err := s.chats.CreateDirectChat(ctx, chatID, userID, targetUserID); err != nil {
		return "", err
	}

	event := map[string]string{
		"chat_id":         chatID,
		"user_id":         userID,
		"target_user_id":  targetUserID,
		"event_type":      "chat.created",
	}
	if err := s.producer.Publish(ctx, s.topic, chatID, event); err != nil {
		s.logger.Error("kafka publish failed", "component", "service.chat", "operation", "create_direct.publish", "chat_id", chatID, "error", err)
		return "", fmt.Errorf("publish chat.created: %w", err)
	}

	return chatID, nil
}

func (s *ChatService) ListForUser(ctx context.Context, userID string) ([]model.ChatListItem, error) {
	return s.chats.ListForUser(ctx, userID)
}
