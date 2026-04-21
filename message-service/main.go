package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	_ "github.com/lib/pq"
)

type ChatMessage struct {
	SenderID string `json:"sender_id"`

	ReceiverID string `json:"receiver_id"`

	Body string `json:"body"`
}

func main() {
	db, err := sql.Open(
		"postgres",
		"postgres://postgres:postgres@postgres:5432/chat?sslmode=disable",
	)

	if err != nil {
		log.Fatal(err)

	}

	reader := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: []string{"kafka:9092"},
			Topic:   "messages",
			GroupID: "message-service",
		},
	)

	for {

		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			continue
		}

		var m ChatMessage

		json.Unmarshal(msg.Value, &m)

		db.Exec(`insert into messages (id, sender_id, receiver_id, body) values ($1,$2,$3,$4)`,
			uuid.New(),
			m.SenderID,
			m.ReceiverID,
			m.Body,
		)

	}

}
