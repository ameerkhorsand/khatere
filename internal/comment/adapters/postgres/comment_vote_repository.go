package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentVoteRepository struct {
	pool *pgxpool.Pool
}

func NewCommentVoteRepository(pool *pgxpool.Pool) *CommentVoteRepository {
	return &CommentVoteRepository{pool: pool}
}

// Upsert relies on the PRIMARY KEY(comment_id, user_id) from
// migration 0016: a second vote from the same user on the same
// comment overwrites the value and created_at in place — a changed
// mind replaces the old vote, it doesn't stack another row.
func (r *CommentVoteRepository) Upsert(ctx context.Context, vote *domain.CommentVote) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO comment_votes (comment_id, user_id, value, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (comment_id, user_id)
		DO UPDATE SET value = EXCLUDED.value, created_at = EXCLUDED.created_at
	`, vote.CommentID, vote.UserID, int(vote.Value), vote.CreatedAt)
	return err
}

func (r *CommentVoteRepository) FindVote(ctx context.Context, commentID, userID uuid.UUID) (*domain.CommentVote, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT comment_id, user_id, value, created_at
		FROM comment_votes
		WHERE comment_id = $1 AND user_id = $2
	`, commentID, userID)

	var v domain.CommentVote
	var value int
	err := row.Scan(&v.CommentID, &v.UserID, &value, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no vote yet is not an error — caller checks for nil
		}
		return nil, err
	}
	v.Value = domain.VoteValue(value)
	return &v, nil
}

// TallyForComments sums up/down votes for every comment in
// commentIDs in one query (avoids one round trip per comment when
// ListCommentsUseCase ranks a whole page of comments).
func (r *CommentVoteRepository) TallyForComments(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID]domain.VoteTally, error) {
	tallies := make(map[uuid.UUID]domain.VoteTally, len(commentIDs))
	if len(commentIDs) == 0 {
		return tallies, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT comment_id,
		       COUNT(*) FILTER (WHERE value = 1)  AS up,
		       COUNT(*) FILTER (WHERE value = -1) AS down
		FROM comment_votes
		WHERE comment_id = ANY($1)
		GROUP BY comment_id
	`, commentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var commentID uuid.UUID
		var tally domain.VoteTally
		if err := rows.Scan(&commentID, &tally.Up, &tally.Down); err != nil {
			return nil, err
		}
		tallies[commentID] = tally
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tallies, nil
}
