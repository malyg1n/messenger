package store

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type PresenceStore struct {
	client *redis.Client
}

func NewPresenceStore(addr string) *PresenceStore {
	return &PresenceStore{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (s *PresenceStore) IsOnline(ctx context.Context, userID string) (bool, error) {
	n, err := s.client.Exists(ctx, s.key(userID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *PresenceStore) Close() error {
	return s.client.Close()
}

func (s *PresenceStore) key(userID string) string {
	return "presence:user:" + userID
}