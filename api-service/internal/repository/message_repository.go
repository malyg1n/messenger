package repository

import (
	"context"
	"database/sql"
	"fmt"

	"api-service/internal/model"
)

// MessageRepository инкапсулирует SQL-запросы для чтения сообщений.
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository создает репозиторий сообщений.
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// ListByChatID возвращает сообщения чата до указанного времени, ограниченные лимитом.
func (r *MessageRepository) ListByChatID(ctx context.Context, chatID string, before string, limit string) ([]model.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		select sender_id, body, created_at
		from messages
		where chat_id = $1
		and ($2 = '' or created_at < $2::timestamp)
		order by created_at desc
		limit $3
	`, chatID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages query: %w", err)
	}
	defer rows.Close()

	messages := make([]model.Message, 0)
	for rows.Next() {
		var message model.Message
		if err := rows.Scan(&message.SenderID, &message.Body, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("list messages scan: %w", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages rows: %w", err)
	}

	return messages, nil
}
