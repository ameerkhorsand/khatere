package postgres

import (
	"context"
	"strconv"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

const messageColumns = `id, hangout_id, sender_id, content, created_at, deleted_at`

func (r *MessageRepository) CreateMessage(ctx context.Context, m *domain.Message) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO hangout_messages (id, hangout_id, sender_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, m.ID, m.HangoutID, m.SenderID, m.Content, m.CreatedAt)
	return err
}

func (r *MessageRepository) ListMessages(ctx context.Context, hangoutID uuid.UUID, filter domain.MessageFilter) ([]domain.Message, error) {
	query := `SELECT ` + messageColumns + ` FROM hangout_messages
		WHERE hangout_id = $1 AND deleted_at IS NULL`
	args := []any{hangoutID}

	if filter.Before != nil {
		query += ` AND created_at < $2`
		args = append(args, *filter.Before)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, filter.Limit)

	rows, err := dbFrom(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func scanMessages(rows pgx.Rows) ([]domain.Message, error) {
	messages := make([]domain.Message, 0)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.HangoutID, &m.SenderID, &m.Content, &m.CreatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}
