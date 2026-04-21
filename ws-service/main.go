package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {

	writer := kafka.Writer{
		Addr:  kafka.TCP("kafka:9092"),
		Topic: "messages",
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			writer.WriteMessages(
				context.Background(),
				kafka.Message{
					Value: msg,
				},
			)
		}
	})

	log.Println("ws started")
	http.ListenAndServe(":8080", nil)
}
