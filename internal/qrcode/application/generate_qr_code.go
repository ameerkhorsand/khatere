package application

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"time"

	"github.com/bLorax/khatere-backend/internal/qrcode/domain"
	"github.com/google/uuid"
)

// ActivityOwnershipChecker is intentionally small so this use case
// doesn't depend on the concrete activity repository — same pattern
// as AttendanceChecker in the rating domain. ActivityRepository.IsOwnedByHost
// (internal/activity/adapters/postgres) satisfies this interface.
type ActivityOwnershipChecker interface {
	// IsOwnedByHost reports whether activityID is a host-sourced
	// activity created by hostAccountID.
	IsOwnedByHost(ctx context.Context, activityID, hostAccountID uuid.UUID) (bool, error)
}

type GenerateQRCodeUseCase struct {
	qrCodes    domain.QRCodeRepository
	activities ActivityOwnershipChecker
}

func NewGenerateQRCodeUseCase(qrCodes domain.QRCodeRepository, activities ActivityOwnershipChecker) *GenerateQRCodeUseCase {
	return &GenerateQRCodeUseCase{qrCodes: qrCodes, activities: activities}
}

type GenerateQRCodeInput struct {
	ActivityID uuid.UUID
	HostID     uuid.UUID
}

// Execute makes the one fixed QR code for a host's activity. Calling
// this twice for the same activity is not an error — it just returns
// the code made the first time, so the host dashboard can call this
// every time it opens without checking first.
func (uc *GenerateQRCodeUseCase) Execute(ctx context.Context, in GenerateQRCodeInput) (*domain.QRCode, error) {
	owned, err := uc.activities.IsOwnedByHost(ctx, in.ActivityID, in.HostID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrNotHostActivity
	}

	existing, err := uc.qrCodes.FindByActivityID(ctx, in.ActivityID)
	if err != nil && err != domain.ErrQRCodeNotFound {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	code, err := randomCode()
	if err != nil {
		return nil, err
	}

	qr := &domain.QRCode{
		ID:         uuid.New(),
		ActivityID: in.ActivityID,
		Code:       code,
		CreatedAt:  time.Now(),
	}
	if err := uc.qrCodes.Create(ctx, qr); err != nil {
		return nil, err
	}
	return qr, nil
}

// randomCode returns a 26-character, unpadded base32 token built
// from 16 random bytes — opaque and unguessable. The frontend draws
// this string as a QR image; the backend never touches image bytes.
func randomCode() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}
