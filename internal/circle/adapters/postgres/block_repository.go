package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockRepository struct {
	pool *pgxpool.Pool
}

func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool}
}

func (r *BlockRepository) Block(
	ctx context.Context,
	blockerID, blockedID uuid.UUID,
) (*domain.Block, error) {
	if blockerID == blockedID {
		return nil, domain.ErrCannotBlockSelf
	}

	var block domain.Block

	err := r.pool.QueryRow(ctx, `
		INSERT INTO circle_blocks (
			blocker_id,
			blocked_id
		)
		VALUES ($1, $2)
		RETURNING
			id,
			blocker_id,
			blocked_id,
			created_at
	`,
		blockerID,
		blockedID,
	).Scan(
		&block.ID,
		&block.BlockerID,
		&block.BlockedID,
		&block.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrAlreadyBlocked
		}

		return nil, err
	}

	return &block, nil
}

func (r *BlockRepository) Unblock(
	ctx context.Context,
	blockerID, blockedID uuid.UUID,
) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM circle_blocks
		WHERE blocker_id = $1
		  AND blocked_id = $2
	`,
		blockerID,
		blockedID,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrBlockNotFound
	}

	return nil
}

func (r *BlockRepository) List(
	ctx context.Context,
	blockerID uuid.UUID,
) ([]domain.Block, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			blocker_id,
			blocked_id,
			created_at
		FROM circle_blocks
		WHERE blocker_id = $1
		ORDER BY created_at DESC, id DESC
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]domain.Block, 0)

	for rows.Next() {
		var block domain.Block

		if err := rows.Scan(
			&block.ID,
			&block.BlockerID,
			&block.BlockedID,
			&block.CreatedAt,
		); err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (r *BlockRepository) IsBlocked(
	ctx context.Context,
	blockerID, blockedID uuid.UUID,
) (bool, error) {
	var exists bool

	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM circle_blocks
			WHERE blocker_id = $1
			  AND blocked_id = $2
		)
	`, blockerID, blockedID).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
