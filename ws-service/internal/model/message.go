package model

import (
	"errors"
	"strings"
)

const MaxMessageBodyLength = 5000

var (
	ErrChatIDRequired  = errors.New("chat_id is required")
	ErrClientMessageIDRequired = errors.New("client_message_id is required")
	ErrBodyRequired    = errors.New("body is required")
	ErrBodyTooLong     = errors.New("body is too long")
)

// ChatMessage — единая модель сообщения в websocket- и kafka-потоке.
type ChatMessage struct {
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
	ClientMessageID string `json:"client_message_id"`
}

// Validate проверяет обязательные поля и ограничение длины сообщения.
func (m ChatMessage) Validate() error {
	if strings.TrimSpace(m.ChatID) == "" {
		return ErrChatIDRequired
	}
	if strings.TrimSpace(m.ClientMessageID) == "" {
		return ErrClientMessageIDRequired
	}
	if strings.TrimSpace(m.Body) == "" {
		return ErrBodyRequired
	}
	if len([]rune(m.Body)) > MaxMessageBodyLength {
		return ErrBodyTooLong
	}
	return nil
}
