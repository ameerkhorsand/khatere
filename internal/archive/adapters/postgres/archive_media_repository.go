package postgres

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArchiveMediaRepository struct {
	pool *pgxpool.Pool
}

func NewArchiveMediaRepository(pool *pgxpool.Pool) *ArchiveMediaRepository {
	return &ArchiveMediaRepository{pool: pool}
}

const archiveMediaColumns = `
	id, archive_id, uploader_id, media_type, storage_key,
	duration_seconds, created_at, deleted_at
`

func (r *ArchiveMediaRepository) Create(ctx context.Context, m *domain.ArchiveMedia) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO archive_media (
			id, archive_id, uploader_id, media_type, storage_key,
			duration_seconds, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.ArchiveID, m.UploaderID, m.MediaType, m.StorageKey,
		m.DurationSeconds, m.CreatedAt)
	return err
}

// ListByArchive returns non-deleted media, oldest first.
func (r *ArchiveMediaRepository) ListByArchive(ctx context.Context, archiveID uuid.UUID) ([]domain.ArchiveMedia, error) {
	rows, err := dbFrom(ctx, r.pool).Query(ctx,
		`SELECT `+archiveMediaColumns+` FROM archive_media
		 WHERE archive_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at ASC`, archiveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArchiveMediaList(rows)
}

func (r *ArchiveMediaRepository) SoftDelete(ctx context.Context, mediaID uuid.UUID) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx,
		`UPDATE archive_media SET deleted_at = now() WHERE id = $1`, mediaID)
	return err
}

func scanArchiveMediaList(rows pgx.Rows) ([]domain.ArchiveMedia, error) {
	media := make([]domain.ArchiveMedia, 0)
	for rows.Next() {
		var m domain.ArchiveMedia
		if err := rows.Scan(&m.ID, &m.ArchiveID, &m.UploaderID, &m.MediaType, &m.StorageKey,
			&m.DurationSeconds, &m.CreatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return media, nil
}
