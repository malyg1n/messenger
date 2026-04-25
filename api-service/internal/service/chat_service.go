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

// ChatService управляет сценариями создания и получения чатов.
type ChatService struct {
	chats  *repository.ChatRepository
	logger *slog.Logger
}

// NewChatService создает сервис чатов.
func NewChatService(chats *repository.ChatRepository, logger *slog.Logger) *ChatService {
	return &ChatService{
		chats:  chats,
		logger: logger,
	}
}

// GetOrCreateDirect возвращает id direct-чата между пользователями, создавая его при отсутствии.
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

	return chatID, nil
}

// ListForUser возвращает список чатов, в которых участвует пользователь.
func (s *ChatService) ListForUser(ctx context.Context, userID string) ([]model.ChatListItem, error) {
	return s.chats.ListForUser(ctx, userID)
}
