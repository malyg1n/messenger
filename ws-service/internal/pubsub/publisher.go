package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"ws-service/internal/model"

	"github.com/redis/go-redis/v9"
)

const MessageChannel = "ws:message"

type Publisher struct {
	client *redis.Client
	logger *slog.Logger
}

func NewPublisher(client *redis.Client, logger *slog.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger,
	}
}

func (p *Publisher) Publish(ctx context.Context, message model.ChatMessage) error {
	value, err := json.Marshal(message)
	if err != nil {
		p.logger.Error("failed to marshal message", "component", "pubsub.publisher", "operation", "publish", "	error", err)
		return fmt.Errorf("marshal message: %w", err)
	}
	if err := p.client.Publish(ctx, MessageChannel, value).Err(); err != nil {
		p.logger.Error("failed to publish message", "component", "pubsub.publisher", "operation", "publish", "error", err)
		return fmt.Errorf("publish message: %w", err)
	}
	p.logger.Info("message published", "component", "pubsub.publisher", "operation", "publish", "message", message)
	return nil
}
