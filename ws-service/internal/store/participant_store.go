package store

import (
	"context"
	"database/sql"
	"fmt"
)

// ParticipantStore читает участников чатов из Postgres.
type ParticipantStore struct {
	db *sql.DB
}

// NewParticipantStore создает store для работы с участниками чатов.
func NewParticipantStore(db *sql.DB) *ParticipantStore {
	return &ParticipantStore{db: db}
}

// GetByChatID возвращает список user_id, участвующих в чате.
func (s *ParticipantStore) GetByChatID(ctx context.Context, chatID string) ([]string, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`
		select user_id
		from chat_participants
		where chat_id = $1
		`,
		chatID,
	)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows.Close()

	users := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		users = append(users, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate participants: %w", err)
	}

	return users, nil
}
