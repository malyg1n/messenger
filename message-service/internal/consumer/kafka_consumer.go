package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"message-service/internal/model"
)

type messageProcessor interface {
	Process(ctx context.Context, message model.ChatMessage) error
}

type KafkaConsumer struct {
	reader    *kafka.Reader
	service   messageProcessor
	logger    *slog.Logger
	topicName string
}

func NewKafkaConsumer(reader *kafka.Reader, service messageProcessor, logger *slog.Logger, topicName string) *KafkaConsumer {
	return &KafkaConsumer{
		reader:    reader,
		service:   service,
		logger:    logger,
		topicName: topicName,
	}
}

func (c *KafkaConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error(
				"failed to read kafka message",
				"component", "consumer",
				"operation", "kafka.fetch_message",
				"topic", c.topicName,
				"error", err,
			)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error(
				"failed to process kafka message",
				"component", "consumer",
				"operation", "kafka.process_message",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", err,
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error(
				"failed to commit kafka message",
				"component", "consumer",
				"operation", "kafka.commit_message",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", err,
			)
		}
	}
}

func (c *KafkaConsumer) processMessage(ctx context.Context, rawMessage kafka.Message) error {
	var message model.ChatMessage
	if err := json.Unmarshal(rawMessage.Value, &message); err != nil {
		return fmt.Errorf("decode message json: %w", err)
	}

	if err := c.service.Process(ctx, message); err != nil {
		return fmt.Errorf("process domain message: %w", err)
	}

	c.logger.Info(
		"message processed",
		"component", "consumer",
		"operation", "kafka.message_processed",
		"topic", rawMessage.Topic,
		"partition", rawMessage.Partition,
		"offset", rawMessage.Offset,
		"chat_id", message.ChatID,
		"sender_id", message.SenderID,
	)

	return nil
}
