package ws

import (
	"context"
	"encoding/json"
	"log/slog"

	"ws-service/internal/broker"
	"ws-service/internal/model"
	"ws-service/internal/store"
)

type Consumer struct {
	brokerConsumer broker.Consumer
	participants   *store.ParticipantStore
	hub            *Hub
	logger         *slog.Logger
}

func NewConsumer(
	brokerConsumer broker.Consumer,
	participants *store.ParticipantStore,
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

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.brokerConsumer.ReadMessage(ctx)
		if err != nil {
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
