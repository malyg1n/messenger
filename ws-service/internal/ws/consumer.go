package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"ws-service/internal/broker"
	"ws-service/internal/model"
	"ws-service/internal/service"
)

// Consumer читает сообщения из Kafka и отправляет их онлайн-участникам чата.
type Consumer struct {
	brokerConsumer broker.Consumer
	participants   *service.ParticipantsService
	hub            *Hub
	logger         *slog.Logger
}

// NewConsumer создает обработчик входящего Kafka-потока для websocket-рассылки.
func NewConsumer(
	brokerConsumer broker.Consumer,
	participants *service.ParticipantsService,
	hub *Hub,
	logger *slog.Logger,
) *Consumer {
	return &Consumer{
		brokerConsumer: brokerConsumer,
		participants:   participants,
		hub:            hub,
		logger:         logger,
	}
}

// Run запускает непрерывный цикл: decode -> resolve participants -> broadcast.
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

		// Получаем состав чата и рассылаем сообщение только его участникам.
		userIDs, err := c.participants.GetByChatID(ctx, chatMessage.ChatID)
		if err != nil {
			c.logger.Error("failed to load chat participants",
				"component", "ws.consumer",
				"operation", "run.load_participants",
				"chat_id", chatMessage.ChatID,
				"error", err,
			)
			continue
		}

		c.hub.Broadcast(userIDs, chatMessage)
	}
}
