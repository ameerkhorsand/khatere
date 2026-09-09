package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArchiveRepository struct {
	pool *pgxpool.Pool
}

func NewArchiveRepository(pool *pgxpool.Pool) *ArchiveRepository {
	return &ArchiveRepository{pool: pool}
}

const archiveColumns = `id, hangout_id, chat_snapshot, status, created_at, updated_at`

func (r *ArchiveRepository) Create(ctx context.Context, a *domain.Archive) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO archives (id, hangout_id, chat_snapshot, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, a.ID, a.HangoutID, a.ChatSnapshot, a.Status, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *ArchiveRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Archive, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+archiveColumns+` FROM archives WHERE id = $1`, id)
	return scanArchive(row)
}

func (r *ArchiveRepository) FindByHangoutID(ctx context.Context, hangoutID uuid.UUID) (*domain.Archive, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+archiveColumns+` FROM archives WHERE hangout_id = $1`, hangoutID)
	return scanArchive(row)
}

func (r *ArchiveRepository) Update(ctx context.Context, a *domain.Archive) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		UPDATE archives
		SET chat_snapshot = $1, status = $2, updated_at = $3
		WHERE id = $4
	`, a.ChatSnapshot, a.Status, a.UpdatedAt, a.ID)
	return err
}

func scanArchive(row pgx.Row) (*domain.Archive, error) {
	var a domain.Archive
	err := row.Scan(&a.ID, &a.HangoutID, &a.ChatSnapshot, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrArchiveNotFound
		}
		return nil, err
	}
	return &a, nil
}
