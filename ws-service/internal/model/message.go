package model

type ChatMessage struct {
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}
