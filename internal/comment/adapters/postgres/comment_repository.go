package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

const commentColumns = `
	id, activity_id, user_id, body, status,
	reviewed_by, reviewed_at, metadata, version,
	created_at, updated_at, deleted_at
`

func (r *CommentRepository) Create(ctx context.Context, c *domain.Comment) error {
	meta, err := json.Marshal(c.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO comments (
			id, activity_id, user_id, body, status,
			metadata, version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.ActivityID, c.UserID, c.Body, c.Status,
		meta, c.Version, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+commentColumns+` FROM comments WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanComment(row)
}

func (r *CommentRepository) ListApprovedByActivity(ctx context.Context, activityID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+commentColumns+`
		FROM comments
		WHERE activity_id = $1 AND status = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, activityID, domain.CommentStatusApproved)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (r *CommentRepository) ListPendingQueue(ctx context.Context) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+commentColumns+`
		FROM comments
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, domain.CommentStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (r *CommentRepository) Update(ctx context.Context, c *domain.Comment) error {
	meta, err := json.Marshal(c.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE comments
		SET body = $1, status = $2, reviewed_by = $3, reviewed_at = $4,
		    metadata = $5, deleted_at = $6
		WHERE id = $7 AND version = $8
	`, c.Body, c.Status, c.ReviewedBy, c.ReviewedAt, meta, c.DeletedAt, c.ID, c.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var c domain.Comment
	var metaBytes []byte
	err := row.Scan(&c.ID, &c.ActivityID, &c.UserID, &c.Body, &c.Status,
		&c.ReviewedBy, &c.ReviewedAt, &metaBytes, &c.Version,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &c.Metadata); err != nil {
		return nil, err
	}
	return &c, nil
}

func scanComments(rows pgx.Rows) ([]domain.Comment, error) {
	comments := make([]domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		var metaBytes []byte
		if err := rows.Scan(&c.ID, &c.ActivityID, &c.UserID, &c.Body, &c.Status,
			&c.ReviewedBy, &c.ReviewedAt, &metaBytes, &c.Version,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metaBytes, &c.Metadata); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}
