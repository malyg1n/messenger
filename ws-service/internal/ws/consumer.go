package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"ws-service/internal/broker"
	"ws-service/internal/model"
	"ws-service/internal/pubsub"
)

// Consumer читает сообщения из Kafka и отправляет их онлайн-участникам чата.
type Consumer struct {
	brokerConsumer broker.Consumer
	logger         *slog.Logger
	publisher      *pubsub.Publisher
}

// NewConsumer создает обработчик входящего Kafka-потока для websocket-рассылки.
func NewConsumer(
	brokerConsumer broker.Consumer,
	logger *slog.Logger,
	publisher *pubsub.Publisher,
) *Consumer {
	return &Consumer{
		brokerConsumer: brokerConsumer,
		logger:         logger,
		publisher:      publisher,
	}
}

// Run запускает непрерывный цикл: read message -> decode -> publish to redis
// then subscribe to redis channel and broadcast to websocket clients
func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Читаем события из Kafka: это уже сообщения, подтвержденные message-service после сохранения в БД.
		msg, err := c.brokerConsumer.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.logger.Error("failed to read kafka message",
				"component", "ws.consumer",
				"operation", "run.read_message",
				"error", err,
			)
			continue
		}

		var chatMessage model.ChatMessage
		if err := json.Unmarshal(msg.Value, &chatMessage); err != nil {
			c.logger.Error("failed to decode kafka message payload",
				"component", "ws.consumer",
				"operation", "run.unmarshal",
				"error", err,
			)
			continue
		}

		c.publisher.Publish(ctx, chatMessage)
	}
}
