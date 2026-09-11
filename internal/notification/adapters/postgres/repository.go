package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, type, recipient_id, actor_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, n.ID, n.Type, n.RecipientID, n.ActorID, n.Metadata, n.CreatedAt)
	return err
}

func (r *Repository) ListForRecipient(ctx context.Context, recipientID uuid.UUID, limit int) ([]domain.Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, recipient_id, actor_id, metadata, read_at, created_at
		FROM notifications
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, recipientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Notification, 0)
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.RecipientID, &n.ActorID, &n.Metadata, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) MarkRead(ctx context.Context, notificationID, recipientID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications
		SET read_at = now()
		WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL
	`, notificationID, recipientID)
	return err
}
