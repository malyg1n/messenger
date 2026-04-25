package service

import (
	"context"
	"fmt"
	"log/slog"
	"ws-service/internal/cache"
	"ws-service/internal/store"
)

// ParticipantsService получает участников чата, используя cache-aside стратегию.
type ParticipantsService struct {
	cache *cache.ParticipantsCache
	store *store.ParticipantStore
	logger *slog.Logger
}

// NewParticipantsService создает сервис участников чатов.
func NewParticipantsService(cache *cache.ParticipantsCache, store *store.ParticipantStore, logger *slog.Logger) *ParticipantsService {
	return &ParticipantsService{cache: cache, store: store, logger: logger}
}	

// GetByChatID сначала ищет участников в кэше, а при промахе читает из БД и кэширует.
func (s *ParticipantsService) GetByChatID(ctx context.Context, chatID string) ([]string, error) {
	participants, err := s.cache.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get participants from cache by chat_id %s: %w", chatID, err)
	}
	if participants != nil {
		s.logger.Info("participants found in cache", "component", "service.participants", "operation", "get_by_chat_id", "chat_id", chatID)
		return participants, nil
	}

	participants, err = s.store.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get participants from store by chat_id %s: %w", chatID, err)
	}
	s.logger.Info("participants loaded from store, setting in cache", "component", "service.participants", "operation", "get_by_chat_id", "chat_id", chatID)
	s.cache.SetByChatID(ctx, chatID, participants)
	return participants, nil
}