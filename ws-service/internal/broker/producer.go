package broker

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Producer задает контракт Kafka-producer для ws-service.
type Producer interface {
	WriteMessage(ctx context.Context, key []byte, value []byte) error
	Ready(ctx context.Context) error
	Close() error
}

// WriterProducer — реализация Producer на базе kafka.Writer.
type WriterProducer struct {
	writer  *kafka.Writer
	brokers []string
}

// NewWriterProducer создает writer для публикации сообщений в Kafka topic.
func NewWriterProducer(brokers []string, topic string) *WriterProducer {
	return &WriterProducer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
		},
		brokers: brokers,
	}
}

// WriteMessage публикует одно сообщение в Kafka.
func (p *WriterProducer) WriteMessage(ctx context.Context, key []byte, value []byte) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// Close освобождает ресурсы Kafka writer.
func (p *WriterProducer) Close() error {
	return p.writer.Close()
}

// Ready проверяет доступность Kafka-брокеров для записи.
func (p *WriterProducer) Ready(ctx context.Context) error {
	return checkBrokers(ctx, p.brokers)
}
