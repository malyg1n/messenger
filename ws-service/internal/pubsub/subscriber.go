package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"ws-service/internal/model"

	"github.com/redis/go-redis/v9"
)

type Subscriber struct {
	client *redis.Client
	logger *slog.Logger
}

func NewSubscriber(client *redis.Client, logger *slog.Logger) *Subscriber {
	return &Subscriber{
		client: client,
		logger: logger,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, handler func(message model.ChatMessage)) error {
	pubsub := s.client.Subscribe(ctx, MessageChannel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	s.logger.Info("subscribed to redis channel", "channel", MessageChannel)

	for {
		select {
		case msg := <-ch:
			var chatMessage model.ChatMessage
			if err := json.Unmarshal([]byte(msg.Payload), &chatMessage); err != nil {
				s.logger.Error("failed to unmarshal message", "error", err)
				continue
			}

			s.logger.Info("received message from redis channel", "channel", MessageChannel, "message", chatMessage)
			handler(chatMessage)

		case <-ctx.Done():
			return ctx.Err()
		}
	}

}
