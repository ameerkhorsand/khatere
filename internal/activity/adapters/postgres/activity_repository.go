package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityRepository struct {
	pool *pgxpool.Pool
}

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}

const selectColumns = `
	id, title, description, source_type, created_by, status,
	reviewed_by, reviewed_at, rejection_reason, metadata, version,
	created_at, updated_at, deleted_at
`

func (r *ActivityRepository) Create(ctx context.Context, a *domain.Activity) error {
	meta, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO activities (
			id, title, description, source_type, created_by, status,
			metadata, version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.Title, a.Description, a.SourceType, a.CreatedBy, a.Status,
		meta, a.Version, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *ActivityRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM activities WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanActivity(row)
}

func (r *ActivityRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Activity, error) {
	query := `SELECT ` + selectColumns + ` FROM activities WHERE deleted_at IS NULL`
	args := []any{}
	argN := 1

	if filter.Status != nil {
		query += ` AND status = $` + strconv.Itoa(argN)
		args = append(args, *filter.Status)
		argN++
	}
	if filter.SourceType != nil {
		query += ` AND source_type = $` + strconv.Itoa(argN)
		args = append(args, *filter.SourceType)
		argN++
	}
	if filter.CreatedBy != nil {
		query += ` AND created_by = $` + strconv.Itoa(argN)
		args = append(args, *filter.CreatedBy)
		argN++
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *ActivityRepository) ListPendingQueue(ctx context.Context) ([]domain.Activity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectColumns+`
		FROM activities
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, domain.ActivityStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *ActivityRepository) Update(ctx context.Context, a *domain.Activity) error {
	meta, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE activities
		SET title = $1, description = $2, status = $3, reviewed_by = $4,
		    reviewed_at = $5, rejection_reason = $6, metadata = $7, deleted_at = $8
		WHERE id = $9 AND version = $10
	`, a.Title, a.Description, a.Status, a.ReviewedBy, a.ReviewedAt,
		a.RejectionReason, meta, a.DeletedAt, a.ID, a.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

// IsOwnedByHost reports whether activityID is a host-sourced
// activity created by hostAccountID. Used by the qrcode and badge
// domains (via their own small ActivityOwnershipChecker interface,
// same pattern as rating's AttendanceChecker) so they don't import
// this package's domain types directly.
func (r *ActivityRepository) IsOwnedByHost(ctx context.Context, activityID, hostAccountID uuid.UUID) (bool, error) {
	var owned bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM activities
			WHERE id = $1 AND created_by = $2
			  AND source_type = 'host' AND deleted_at IS NULL
		)
	`, activityID, hostAccountID).Scan(&owned)
	return owned, err
}

func scanActivity(row pgx.Row) (*domain.Activity, error) {
	var a domain.Activity
	var metaBytes []byte
	err := row.Scan(&a.ID, &a.Title, &a.Description, &a.SourceType, &a.CreatedBy, &a.Status,
		&a.ReviewedBy, &a.ReviewedAt, &a.RejectionReason, &metaBytes, &a.Version,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrActivityNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &a.Metadata); err != nil {
		return nil, err
	}
	return &a, nil
}

func scanActivities(rows pgx.Rows) ([]domain.Activity, error) {
	activities := make([]domain.Activity, 0)
	for rows.Next() {
		var a domain.Activity
		var metaBytes []byte
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.SourceType, &a.CreatedBy, &a.Status,
			&a.ReviewedBy, &a.ReviewedAt, &a.RejectionReason, &metaBytes, &a.Version,
			&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metaBytes, &a.Metadata); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activities, nil
}
