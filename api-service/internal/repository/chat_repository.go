package repository

import (
	"context"
	"database/sql"
	"fmt"

	"api-service/internal/model"
)

// ChatRepository инкапсулирует SQL-операции для таблиц чатов.
type ChatRepository struct {
	db *sql.DB
}

// NewChatRepository создает репозиторий чатов на базе подключения к БД.
func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// FindDirectChatID ищет существующий direct-чат между двумя пользователями.
func (r *ChatRepository) FindDirectChatID(ctx context.Context, userID string, targetUserID string) (string, error) {
	var chatID string
	err := r.db.QueryRowContext(ctx, `
		select c.id
		from chats c
		join chat_participants p1 on p1.chat_id = c.id
		join chat_participants p2 on p2.chat_id = c.id
		where p1.user_id = $1 and p2.user_id = $2
		limit 1
	`, userID, targetUserID).Scan(&chatID)
	if err != nil {
		return chatID, fmt.Errorf("find direct chat id: %w", err)
	}
	return chatID, nil
}

// CreateDirectChat создает новый чат и добавляет обоих участников.
func (r *ChatRepository) CreateDirectChat(ctx context.Context, chatID string, userID string, targetUserID string) error {
	_, err := r.db.ExecContext(ctx, "insert into chats(id) values ($1)", chatID)
	if err != nil {
		return fmt.Errorf("create chat: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		insert into chat_participants (chat_id, user_id)
		values ($1,$2),($1,$3)
	`, chatID, userID, targetUserID)
	if err != nil {
		return fmt.Errorf("add chat participants: %w", err)
	}
	return nil
}

// ListForUser возвращает все чаты пользователя и метаданные последнего сообщения.
func (r *ChatRepository) ListForUser(ctx context.Context, userID string) ([]model.ChatListItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		select
			c.id as chat_id,
			u.username as title,
			coalesce(m.body, '') as last_message,
			coalesce(m.created_at::text, '') as last_message_at
		from chats c
		join chat_participants self_p on self_p.chat_id = c.id
		join chat_participants other_p on other_p.chat_id = c.id and other_p.user_id <> self_p.user_id
		join users u on u.id = other_p.user_id
		left join lateral (
			select body, created_at
			from messages
			where chat_id = c.id
			order by created_at desc
			limit 1
		) m on true
		where self_p.user_id = $1
		order by m.created_at desc nulls last
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list chats query: %w", err)
	}
	defer rows.Close()

	chats := make([]model.ChatListItem, 0)
	for rows.Next() {
		var item model.ChatListItem
		if err := rows.Scan(&item.ChatID, &item.Title, &item.LastMessage, &item.LastMessageAt); err != nil {
			return nil, fmt.Errorf("list chats scan: %w", err)
		}
		chats = append(chats, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list chats rows: %w", err)
	}

	return chats, nil
}
