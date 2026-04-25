package cache

import (
	"context"
	"sync"
)

// ParticipantsCache хранит в памяти кэш участников чатов.
type ParticipantsCache struct {
	participants map[string][]string
	mutex sync.RWMutex
}

// NewParticipantsCache создает пустой in-memory кэш участников.
func NewParticipantsCache() *ParticipantsCache {
	return &ParticipantsCache{
		participants: make(map[string][]string),
		mutex: sync.RWMutex{},
	}
}

// GetByChatID возвращает участников чата из кэша, если они уже загружены.
func (c *ParticipantsCache) GetByChatID(ctx context.Context, chatID string) ([]string, error) {
	c.mutex.RLock()
	participants, ok := c.participants[chatID]
	c.mutex.RUnlock()
	if ok {
		return participants, nil
	}
	return nil, nil
}

// SetByChatID сохраняет состав участников чата в кэш.
func (c *ParticipantsCache) SetByChatID(ctx context.Context, chatID string, participants []string) {
	c.mutex.Lock()
	c.participants[chatID] = participants
	c.mutex.Unlock()
}