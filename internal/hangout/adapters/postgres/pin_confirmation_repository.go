package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PinConfirmationRepository struct {
	pool *pgxpool.Pool
}

func NewPinConfirmationRepository(pool *pgxpool.Pool) *PinConfirmationRepository {
	return &PinConfirmationRepository{pool: pool}
}

const pinConfirmationColumns = `
	id, pin_id, user_id, status, confirmed_at, created_at, updated_at
`

// ResetForPin clears every existing confirmation row for pinID and
// inserts one fresh 'pending' row per user given. Call it inside the
// same transaction as the pin write that triggered the reset, so the
// pin and its confirmation set change together or not at all.
func (r *PinConfirmationRepository) ResetForPin(ctx context.Context, pinID uuid.UUID, userIDs []uuid.UUID) error {
	db := dbFrom(ctx, r.pool)

	if _, err := db.Exec(ctx, `DELETE FROM hangout_meetup_pin_confirmations WHERE pin_id = $1`, pinID); err != nil {
		return err
	}

	for _, userID := range userIDs {
		if _, err := db.Exec(ctx, `
			INSERT INTO hangout_meetup_pin_confirmations (id, pin_id, user_id, status)
			VALUES ($1, $2, $3, 'pending')
		`, uuid.New(), pinID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PinConfirmationRepository) FindConfirmation(ctx context.Context, pinID, userID uuid.UUID) (*domain.PinConfirmation, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+pinConfirmationColumns+` FROM hangout_meetup_pin_confirmations WHERE pin_id = $1 AND user_id = $2`,
		pinID, userID)
	return scanPinConfirmation(row)
}

func (r *PinConfirmationRepository) ListByPin(ctx context.Context, pinID uuid.UUID) ([]domain.PinConfirmation, error) {
	rows, err := dbFrom(ctx, r.pool).Query(ctx,
		`SELECT `+pinConfirmationColumns+` FROM hangout_meetup_pin_confirmations WHERE pin_id = $1 ORDER BY created_at ASC`,
		pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPinConfirmations(rows)
}

func (r *PinConfirmationRepository) UpdateConfirmation(ctx context.Context, c *domain.PinConfirmation) error {
	tag, err := dbFrom(ctx, r.pool).Exec(ctx, `
		UPDATE hangout_meetup_pin_confirmations
		SET status = $1, confirmed_at = $2, updated_at = $3
		WHERE id = $4
	`, c.Status, c.ConfirmedAt, c.UpdatedAt, c.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPinConfirmationNotFound
	}
	return nil
}

func scanPinConfirmation(row pgx.Row) (*domain.PinConfirmation, error) {
	var c domain.PinConfirmation
	err := row.Scan(&c.ID, &c.PinID, &c.UserID, &c.Status, &c.ConfirmedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPinConfirmationNotFound
		}
		return nil, err
	}
	return &c, nil
}

func scanPinConfirmations(rows pgx.Rows) ([]domain.PinConfirmation, error) {
	confirmations := make([]domain.PinConfirmation, 0)
	for rows.Next() {
		var c domain.PinConfirmation
		if err := rows.Scan(&c.ID, &c.PinID, &c.UserID, &c.Status, &c.ConfirmedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		confirmations = append(confirmations, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return confirmations, nil
}
