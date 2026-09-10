package postgres

import (
	"context"
	"errors"

	badgeDomain "github.com/bLorax/khatere-backend/internal/badge/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BadgeLookupRepository is a purpose-built adapter for the
// userbadge domain's BadgeLookup interface — same reasoning as
// the attendance domain's ResolveCode on the qrcode Postgres
// adapter, so this package never needs to import badge/domain
// outside of this one file.
type BadgeLookupRepository struct {
	pool *pgxpool.Pool
}

func NewBadgeLookupRepository(pool *pgxpool.Pool) *BadgeLookupRepository {
	return &BadgeLookupRepository{pool: pool}
}

func (r *BadgeLookupRepository) FindByActivityID(ctx context.Context, activityID uuid.UUID) (badgeID uuid.UUID, name, iconKey string, found bool, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT id, name, icon_key FROM badges WHERE activity_id = $1`, activityID,
	).Scan(&badgeID, &name, &iconKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, "", "", false, nil
		}
		return uuid.UUID{}, "", "", false, err
	}
	return badgeID, name, iconKey, true, nil
}

// _ ensures this file compiles against the real badge error type
// even though the interface itself never mentions it.
var _ = badgeDomain.ErrBadgeNotFound
