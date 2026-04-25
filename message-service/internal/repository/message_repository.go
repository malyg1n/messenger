package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"message-service/internal/model"
)

// MessageRepository сохраняет сообщения в хранилище Postgres.
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository создает репозиторий сообщений.
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Save сохраняет одно сообщение чата в таблицу messages.
func (r *MessageRepository) Save(ctx context.Context, message model.ChatMessage) error {
	_, err := r.db.ExecContext(
		ctx,
		`insert into messages (id, sender_id, chat_id, body) values ($1, $2, $3, $4)`,
		uuid.New(),
		message.SenderID,
		message.ChatID,
		message.Body,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	return nil
}
