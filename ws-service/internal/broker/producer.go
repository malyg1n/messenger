package broker

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	WriteMessage(ctx context.Context, key []byte, value []byte) error
	Close() error
}

type WriterProducer struct {
	writer *kafka.Writer
}

func NewWriterProducer(brokers []string, topic string) *WriterProducer {
	return &WriterProducer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
		},
	}
}

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

func (p *WriterProducer) Close() error {
	return p.writer.Close()
}
