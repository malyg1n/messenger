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
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

func main() {
	log.Println("Starting message service")
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
			log.Println("error reading message", err)
			continue
		}

		var m ChatMessage

		json.Unmarshal(msg.Value, &m)
		log.Println("message", m)

		db.Exec(`insert into messages (id, sender_id, chat_id, body) values ($1,$2,$3,$4)`,
			uuid.New(),
			m.SenderID,
			m.ChatID,
			m.Body,
		)

	}

}
