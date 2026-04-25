package broker

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

var (
	ErrNoKafkaBrokers  = errors.New("no kafka brokers configured")
	ErrKafkaUnavailable = errors.New("kafka brokers are unavailable")
)

// checkBrokers проверяет, что доступен хотя бы один Kafka broker.
func checkBrokers(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return ErrNoKafkaBrokers
	}

	var lastErr error
	for _, brokerAddr := range brokers {
		conn, err := kafka.DialContext(ctx, "tcp", brokerAddr)
		if err != nil {
			lastErr = err
			continue
		}
		_ = conn.Close()
		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("%w: %w", ErrKafkaUnavailable, lastErr)
	}
	return ErrKafkaUnavailable
}
