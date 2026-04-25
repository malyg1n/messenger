package broker

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Producer публикует обработанные сообщения в Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer создает Kafka writer для указанного topic.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
		},
	}
}

// Publish отправляет одно сообщение в Kafka.
func (p *Producer) Publish(ctx context.Context, key []byte, value []byte) error {
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	}); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	return nil
}

// Close освобождает ресурсы Kafka writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
