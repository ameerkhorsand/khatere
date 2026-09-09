package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeetupPinRepository struct {
	pool *pgxpool.Pool
}

func NewMeetupPinRepository(pool *pgxpool.Pool) *MeetupPinRepository {
	return &MeetupPinRepository{pool: pool}
}

const meetupPinColumns = `
	id, hangout_id, place_name, address, latitude, longitude,
	scheduled_at, proposed_by, version, created_at, updated_at
`

func (r *MeetupPinRepository) Create(ctx context.Context, p *domain.MeetupPin) error {
	_, err := dbFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO hangout_meetup_pins (
			id, hangout_id, place_name, address, latitude, longitude,
			scheduled_at, proposed_by, version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, p.ID, p.HangoutID, p.PlaceName, p.Address, p.Latitude, p.Longitude,
		p.ScheduledAt, p.ProposedBy, p.Version, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *MeetupPinRepository) FindByHangoutID(ctx context.Context, hangoutID uuid.UUID) (*domain.MeetupPin, error) {
	row := dbFrom(ctx, r.pool).QueryRow(ctx,
		`SELECT `+meetupPinColumns+` FROM hangout_meetup_pins WHERE hangout_id = $1`, hangoutID)
	return scanMeetupPin(row)
}

func (r *MeetupPinRepository) Update(ctx context.Context, p *domain.MeetupPin) error {
	tag, err := dbFrom(ctx, r.pool).Exec(ctx, `
		UPDATE hangout_meetup_pins
		SET place_name = $1, address = $2, latitude = $3, longitude = $4,
		    scheduled_at = $5, proposed_by = $6
		WHERE id = $7 AND version = $8
	`, p.PlaceName, p.Address, p.Latitude, p.Longitude, p.ScheduledAt, p.ProposedBy, p.ID, p.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPinVersionConflict
	}
	return nil
}

func scanMeetupPin(row pgx.Row) (*domain.MeetupPin, error) {
	var p domain.MeetupPin
	err := row.Scan(&p.ID, &p.HangoutID, &p.PlaceName, &p.Address, &p.Latitude, &p.Longitude,
		&p.ScheduledAt, &p.ProposedBy, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPinNotFound
		}
		return nil, err
	}
	return &p, nil
}
