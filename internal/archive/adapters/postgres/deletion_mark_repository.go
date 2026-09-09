package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeletionMarkRepository struct {
	pool *pgxpool.Pool
}

func NewDeletionMarkRepository(pool *pgxpool.Pool) *DeletionMarkRepository {
	return &DeletionMarkRepository{pool: pool}
}

func (r *DeletionMarkRepository) Add(ctx context.Context, mark *domain.ArchiveDeletionMark) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO archive_deletion_marks (archive_id, user_id, created_at)
		VALUES ($1, $2, $3)
	`, mark.ArchiveID, mark.UserID, mark.CreatedAt)
	if err != nil {
		if uniqueViolation(err) {
			return domain.ErrAlreadyMarkedDeleted
		}
		return err
	}
	return nil
}

func (r *DeletionMarkRepository) Count(ctx context.Context, archiveID uuid.UUID) (int, error) {
	var count int
	err := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT COUNT(*) FROM archive_deletion_marks WHERE archive_id = $1`, archiveID,
	).Scan(&count)
	return count, err
}

// uniqueViolation reports whether err is a Postgres unique/primary-key
// violation (code 23505) — here, that means the user already has a
// mark on this archive (archive_id, user_id is the primary key).
func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
