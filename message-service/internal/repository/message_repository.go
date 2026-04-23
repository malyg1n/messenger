package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"message-service/internal/model"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

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
