package broker

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Consumer interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

type ReaderConsumer struct {
	reader *kafka.Reader
}

func NewReaderConsumer(brokers []string, topic string, groupID string) *ReaderConsumer {
	return &ReaderConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

func (c *ReaderConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("read kafka message: %w", err)
	}
	return msg, nil
}

func (c *ReaderConsumer) Close() error {
	return c.reader.Close()
}
