package repository

import (
	"context"
	"database/sql"

	"api-service/internal/model"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) ListByChatID(ctx context.Context, chatID string, limit string) ([]model.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		select sender_id, body, created_at
		from messages
		where chat_id = $1
		order by created_at desc
		limit $2
	`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]model.Message, 0)
	for rows.Next() {
		var message model.Message
		if err := rows.Scan(&message.SenderID, &message.Body, &message.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
