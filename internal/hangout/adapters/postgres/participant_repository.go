package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ParticipantRepository struct {
	pool *pgxpool.Pool
}

func NewParticipantRepository(pool *pgxpool.Pool) *ParticipantRepository {
	return &ParticipantRepository{pool: pool}
}

const participantColumns = `
	id, hangout_id, user_id, role, invite_status, reason,
	invited_by, invited_at, responded_at, created_at, updated_at
`

// uniqueViolation reports whether err is a Postgres unique-constraint
// error (code 23505) — here, that means a (hangout_id, user_id) row
// already exists.
func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *ParticipantRepository) AddParticipant(ctx context.Context, p *domain.Participant) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO hangout_participants (
			id, hangout_id, user_id, role, invite_status, reason,
			invited_by, invited_at, responded_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, p.ID, p.HangoutID, p.UserID, p.Role, p.InviteStatus, p.Reason,
		p.InvitedBy, p.InvitedAt, p.RespondedAt, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		if uniqueViolation(err) {
			return domain.ErrAlreadyParticipant
		}
		return err
	}
	return nil
}

func (r *ParticipantRepository) FindParticipant(ctx context.Context, hangoutID, userID uuid.UUID) (*domain.Participant, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+participantColumns+` FROM hangout_participants WHERE hangout_id = $1 AND user_id = $2`,
		hangoutID, userID)
	return scanParticipant(row)
}

func (r *ParticipantRepository) ListParticipants(ctx context.Context, hangoutID uuid.UUID) ([]domain.Participant, error) {
	rows, err := dbFrom(ctx, r.pool).Query(ctx,
		`SELECT `+participantColumns+` FROM hangout_participants WHERE hangout_id = $1 ORDER BY invited_at ASC`,
		hangoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanParticipants(rows)
}

func (r *ParticipantRepository) CountActive(ctx context.Context, hangoutID uuid.UUID) (int, error) {
	var count int
	err := dbFrom(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM hangout_participants
		WHERE hangout_id = $1 AND invite_status IN ('pending', 'accepted')
	`, hangoutID).Scan(&count)
	return count, err
}

func (r *ParticipantRepository) UpdateParticipant(ctx context.Context, p *domain.Participant) error {
	// The WHERE clause only matches a still-pending row, so a second
	// response arriving concurrently (e.g. two taps on Accept/Reject)
	// affects zero rows instead of overwriting the first answer.
	tag, err := dbFrom(ctx, r.pool).Exec(ctx, `
		UPDATE hangout_participants
		SET invite_status = $1, reason = $2, responded_at = $3, updated_at = $4
		WHERE id = $5 AND invite_status = 'pending'
	`, p.InviteStatus, p.Reason, p.RespondedAt, p.UpdatedAt, p.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInviteAlreadyAnswered
	}
	return nil
}

func scanParticipant(row pgx.Row) (*domain.Participant, error) {
	var p domain.Participant
	err := row.Scan(&p.ID, &p.HangoutID, &p.UserID, &p.Role, &p.InviteStatus, &p.Reason,
		&p.InvitedBy, &p.InvitedAt, &p.RespondedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrParticipantNotFound
		}
		return nil, err
	}
	return &p, nil
}

func scanParticipants(rows pgx.Rows) ([]domain.Participant, error) {
	participants := make([]domain.Participant, 0)
	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(&p.ID, &p.HangoutID, &p.UserID, &p.Role, &p.InviteStatus, &p.Reason,
			&p.InvitedBy, &p.InvitedAt, &p.RespondedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return participants, nil
}
