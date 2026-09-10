package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/attendance/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceVerificationRepository struct {
	pool *pgxpool.Pool
}

func NewAttendanceVerificationRepository(pool *pgxpool.Pool) *AttendanceVerificationRepository {
	return &AttendanceVerificationRepository{pool: pool}
}

const verificationColumns = `id, activity_id, user_id, qr_code_id, verified_at`

func (r *AttendanceVerificationRepository) Create(ctx context.Context, v *domain.AttendanceVerification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO attendance_verifications (id, activity_id, user_id, qr_code_id, verified_at)
		VALUES ($1, $2, $3, $4, $5)
	`, v.ID, v.ActivityID, v.UserID, v.QRCodeID, v.VerifiedAt)
	return err
}

func (r *AttendanceVerificationRepository) FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*domain.AttendanceVerification, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+verificationColumns+` FROM attendance_verifications WHERE activity_id = $1 AND user_id = $2`,
		activityID, userID)
	return scanVerification(row)
}

// IsVerified is a purpose-built read for the comment domain's
// ranking boost (Phase 7 Step 7) — same reasoning as ResolveCode
// in this package's qr_code_repository.go. It returns a plain
// bool so comment/application never needs to import
// attendance/domain.
func (r *AttendanceVerificationRepository) IsVerified(ctx context.Context, activityID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM attendance_verifications WHERE activity_id = $1 AND user_id = $2)`,
		activityID, userID,
	).Scan(&exists)
	return exists, err
}

func scanVerification(row pgx.Row) (*domain.AttendanceVerification, error) {
	var v domain.AttendanceVerification
	err := row.Scan(&v.ID, &v.ActivityID, &v.UserID, &v.QRCodeID, &v.VerifiedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVerificationNotFound
		}
		return nil, err
	}
	return &v, nil
}
