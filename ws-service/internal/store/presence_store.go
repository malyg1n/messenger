package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewPresenceStore(client *redis.Client, ttl time.Duration) *PresenceStore {
	return &PresenceStore{
		client: client,
		ttl: ttl,
	}
}

func (s *PresenceStore) SetOnline(ctx context.Context, userID string) error {
	return s.client.Set(ctx, s.key(userID), "online", s.ttl).Err()
}

func (s *PresenceStore) RefreshOnline(ctx context.Context, userID string) error {
	return s.client.Expire(ctx, s.key(userID), s.ttl).Err()
}

func (s *PresenceStore) SetOffline(ctx context.Context, userID string) error {
	return s.client.Del(ctx, s.key(userID)).Err()
}

func (s *PresenceStore) Close() error {
	return s.client.Close()
}

func (s *PresenceStore) key(userID string) string {
	return "presence:user:" + userID
}