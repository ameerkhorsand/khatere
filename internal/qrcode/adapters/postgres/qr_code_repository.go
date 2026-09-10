package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/qrcode/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QRCodeRepository struct {
	pool *pgxpool.Pool
}

func NewQRCodeRepository(pool *pgxpool.Pool) *QRCodeRepository {
	return &QRCodeRepository{pool: pool}
}

const qrCodeColumns = `id, activity_id, code, created_at`

func (r *QRCodeRepository) Create(ctx context.Context, qr *domain.QRCode) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO qr_codes (id, activity_id, code, created_at)
		VALUES ($1, $2, $3, $4)
	`, qr.ID, qr.ActivityID, qr.Code, qr.CreatedAt)
	return err
}

func (r *QRCodeRepository) FindByActivityID(ctx context.Context, activityID uuid.UUID) (*domain.QRCode, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+qrCodeColumns+` FROM qr_codes WHERE activity_id = $1`, activityID)
	return scanQRCode(row)
}

func (r *QRCodeRepository) FindByCode(ctx context.Context, code string) (*domain.QRCode, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+qrCodeColumns+` FROM qr_codes WHERE code = $1`, code)
	return scanQRCode(row)
}

func scanQRCode(row pgx.Row) (*domain.QRCode, error) {
	var qr domain.QRCode
	err := row.Scan(&qr.ID, &qr.ActivityID, &qr.Code, &qr.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQRCodeNotFound
		}
		return nil, err
	}
	return &qr, nil
}
