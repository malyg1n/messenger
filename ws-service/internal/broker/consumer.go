package broker

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Consumer задает контракт Kafka-consumer для ws-service.
type Consumer interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Ready(ctx context.Context) error
	Close() error
}

// ReaderConsumer — реализация Consumer на базе kafka.Reader.
type ReaderConsumer struct {
	reader  *kafka.Reader
	brokers []string
}

// NewReaderConsumer создает Kafka reader с настройками topic/group.
func NewReaderConsumer(brokers []string, topic string, groupID string) *ReaderConsumer {
	return &ReaderConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		brokers: brokers,
	}
}

// ReadMessage читает одно сообщение из Kafka.
func (c *ReaderConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("read kafka message: %w", err)
	}
	return msg, nil
}

// Close освобождает ресурсы Kafka reader.
func (c *ReaderConsumer) Close() error {
	return c.reader.Close()
}

// Ready проверяет доступность Kafka-брокеров для чтения.
func (c *ReaderConsumer) Ready(ctx context.Context) error {
	return checkBrokers(ctx, c.brokers)
}
