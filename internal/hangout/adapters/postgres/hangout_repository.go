package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HangoutRepository struct {
	pool *pgxpool.Pool
}

func NewHangoutRepository(pool *pgxpool.Pool) *HangoutRepository {
	return &HangoutRepository{pool: pool}
}

const hangoutColumns = `
	id, activity_id, organizer_id, title, description, status, scheduled_at,
	scheduled_end_at, metadata, version, created_at, updated_at, deleted_at
`

func (r *HangoutRepository) Create(ctx context.Context, h *domain.Hangout) error {
	meta, err := json.Marshal(h.Metadata)
	if err != nil {
		return err
	}
	_, err = dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO hangouts (
			id, activity_id, organizer_id, title, description, status, scheduled_at,
			scheduled_end_at, metadata, version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, h.ID, h.ActivityID, h.OrganizerID, h.Title, h.Description, h.Status, h.ScheduledAt,
		h.ScheduledEndAt, meta, h.Version, h.CreatedAt, h.UpdatedAt)
	return err
}

// Attended reports whether userID has an accepted invite on a
// completed hangout linked to activityID. Implements the
// rating/application.AttendanceChecker interface — see that file
// for why this query lives here rather than in the rating package.
func (r *HangoutRepository) Attended(ctx context.Context, activityID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := dbFrom(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM hangouts h
			JOIN hangout_participants p ON p.hangout_id = h.id
			WHERE h.activity_id = $1
			  AND p.user_id = $2
			  AND h.status = 'completed'
			  AND p.invite_status = 'accepted'
			  AND h.deleted_at IS NULL
		)
	`, activityID, userID).Scan(&exists)
	return exists, err
}

func (r *HangoutRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Hangout, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+hangoutColumns+` FROM hangouts WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanHangout(row)
}

func (r *HangoutRepository) ListByUser(ctx context.Context, userID uuid.UUID, filter domain.ListFilter) ([]domain.Hangout, error) {
	query := `
		SELECT ` + hangoutColumns + `
		FROM hangouts h
		WHERE h.deleted_at IS NULL
		  AND EXISTS (
			SELECT 1 FROM hangout_participants p
			WHERE p.hangout_id = h.id
			  AND p.user_id = $1
			  AND p.invite_status IN ('pending', 'accepted')
		  )
	`
	args := []any{userID}

	if filter.Status != nil {
		query += ` AND h.status = $` + strconv.Itoa(len(args)+1)
		args = append(args, *filter.Status)
	}
	query += ` ORDER BY h.scheduled_at DESC NULLS LAST, h.created_at DESC`

	rows, err := dbFrom(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHangouts(rows)
}

func (r *HangoutRepository) Update(ctx context.Context, h *domain.Hangout) error {
	meta, err := json.Marshal(h.Metadata)
	if err != nil {
		return err
	}
	tag, err := dbFrom(ctx, r.pool).Exec(ctx, `
		UPDATE hangouts
		SET title = $1, description = $2, status = $3, scheduled_at = $4,
		    scheduled_end_at = $5, metadata = $6, deleted_at = $7
		WHERE id = $8 AND version = $9
	`, h.Title, h.Description, h.Status, h.ScheduledAt, h.ScheduledEndAt,
		meta, h.DeletedAt, h.ID, h.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func scanHangout(row pgx.Row) (*domain.Hangout, error) {
	var h domain.Hangout
	var metaBytes []byte
	err := row.Scan(&h.ID, &h.ActivityID, &h.OrganizerID, &h.Title, &h.Description, &h.Status, &h.ScheduledAt,
		&h.ScheduledEndAt, &metaBytes, &h.Version, &h.CreatedAt, &h.UpdatedAt, &h.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHangoutNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &h.Metadata); err != nil {
		return nil, err
	}
	return &h, nil
}

func scanHangouts(rows pgx.Rows) ([]domain.Hangout, error) {
	hangouts := make([]domain.Hangout, 0)
	for rows.Next() {
		var h domain.Hangout
		var metaBytes []byte
		if err := rows.Scan(&h.ID, &h.ActivityID, &h.OrganizerID, &h.Title, &h.Description, &h.Status, &h.ScheduledAt,
			&h.ScheduledEndAt, &metaBytes, &h.Version, &h.CreatedAt, &h.UpdatedAt, &h.DeletedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metaBytes, &h.Metadata); err != nil {
			return nil, err
		}
		hangouts = append(hangouts, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hangouts, nil
}
