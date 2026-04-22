package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
	Close() error
}

type WriterProducer struct {
	writer *kafkago.Writer
}

func NewWriterProducer(brokers []string, clientID string) *WriterProducer {
	return &WriterProducer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
			Async:        false,
			Transport: &kafkago.Transport{
				ClientID: clientID,
			},
		},
	}
}

func (p *WriterProducer) Publish(ctx context.Context, topic string, key string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal kafka payload: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: body,
	})
	if err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

func (p *WriterProducer) Close() error {
	return p.writer.Close()
}
