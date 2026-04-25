package model

import (
	"errors"
	"strings"
)

const MaxMessageBodyLength = 5000

var (
	ErrChatIDRequired   = errors.New("chat_id is required")
	ErrSenderIDRequired = errors.New("sender_id is required")
	ErrBodyRequired     = errors.New("body is required")
	ErrBodyTooLong      = errors.New("body is too long")
)

// ChatMessage — доменная модель сообщения, полученного из Kafka.
type ChatMessage struct {
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

// Validate проверяет обязательные поля и ограничение длины текста сообщения.
func (m ChatMessage) Validate() error {
	if strings.TrimSpace(m.ChatID) == "" {
		return ErrChatIDRequired
	}
	if strings.TrimSpace(m.SenderID) == "" {
		return ErrSenderIDRequired
	}
	if strings.TrimSpace(m.Body) == "" {
		return ErrBodyRequired
	}
	if len([]rune(m.Body)) > MaxMessageBodyLength {
		return ErrBodyTooLong
	}
	return nil
}
