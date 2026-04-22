package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"

	"database/sql"

	_ "github.com/lib/pq"
)

type ChatMessage struct {
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var connections = struct {
	sync.RWMutex
	m map[string]*websocket.Conn
}{
	m: make(map[string]*websocket.Conn),
}

func main() {

	writer := kafka.Writer{
		Addr:                   kafka.TCP("kafka:9092"),
		Topic:                  "messages",
		AllowAutoTopicCreation: true,
	}

	reader := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: []string{"kafka:9092"},
			Topic:   "messages",
			GroupID: "ws-service",
		},
	)

	db, err := sql.Open(
		"postgres",
		"postgres://postgres:postgres@postgres:5432/chat?sslmode=disable",
	)

	if err != nil {
		log.Fatal(err)
	}

	go consumeMessages(reader, db)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		userID := r.URL.Query().Get("user_id")

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			return
		}

		connections.Lock()
		connections.m[userID] = conn
		connections.Unlock()

		for {

			_, msg, err := conn.ReadMessage()

			if err != nil {

				connections.Lock()
				delete(connections.m, userID)
				connections.Unlock()

				break
			}

			var m ChatMessage
			json.Unmarshal(msg, &m)

			writer.WriteMessages(
				context.Background(),
				kafka.Message{
					Key:   []byte(m.ChatID),
					Value: msg,
				},
			)
		}

	})

	log.Println("ws started")

	http.ListenAndServe(":8080", nil)

}

func consumeMessages(
	reader *kafka.Reader,
	db *sql.DB,
) {

	for {

		msg, err := reader.ReadMessage(context.Background())

		if err != nil {
			continue
		}

		var m ChatMessage

		json.Unmarshal(msg.Value, &m)

		broadcast(db, m)

	}

}

func broadcast(db *sql.DB, m ChatMessage) {

	connections.RLock()

	defer connections.RUnlock()

	users := getParticipants(
		db,
		m.ChatID,
	)

	for _, uid := range users {

		conn, ok := connections.m[uid]

		if ok {

			conn.WriteJSON(m)

		}

	}

}

func getParticipants(
	db *sql.DB,
	chatID string,
) []string {

	rows, err := db.Query(
		`
		select user_id
		from chat_participants
		where chat_id = $1
		`,
		chatID,
	)

	if err != nil {
		return nil
	}

	defer rows.Close()

	var users []string

	for rows.Next() {

		var id string

		rows.Scan(&id)

		users = append(users, id)

	}

	return users

}
