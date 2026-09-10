package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/attendance/domain"
	"github.com/google/uuid"
)

// QRCodeLookup is intentionally small so this use case doesn't
// depend on the concrete qrcode repository, and never needs to
// import qrcode/domain — same reasoning as rating's AttendanceChecker.
// A purpose-built ResolveCode method on qrcodePG.QRCodeRepository
// (internal/qrcode/adapters/postgres) satisfies this interface.
type QRCodeLookup interface {
	// ResolveCode reports the activity and QR code id a scanned
	// code belongs to. found is false with a nil err when the code
	// is simply unrecognized — that's a normal bad scan, not a
	// system error.
	ResolveCode(ctx context.Context, code string) (activityID, qrCodeID uuid.UUID, found bool, err error)
}

// BadgeAwarder is the one place the attendance module reaches across
// into the userbadge module — the mirror of hangout's archivecreator
// gateway. Wired up once the userbadge domain exists (Phase 7 Step 5).
type BadgeAwarder interface {
	// AwardIfBadgeExists gives userID the badge set up for
	// activityID, if the host has set one up. A no-op, not an
	// error, when the activity has no badge.
	AwardIfBadgeExists(ctx context.Context, userID, activityID uuid.UUID) error
}

type VerifyScanUseCase struct {
	verifications domain.AttendanceVerificationRepository
	qrCodes       QRCodeLookup
	badges        BadgeAwarder
}

func NewVerifyScanUseCase(
	verifications domain.AttendanceVerificationRepository,
	qrCodes QRCodeLookup,
	badges BadgeAwarder,
) *VerifyScanUseCase {
	return &VerifyScanUseCase{verifications: verifications, qrCodes: qrCodes, badges: badges}
}

type VerifyScanInput struct {
	UserID      uuid.UUID
	ScannedCode string
}

// Execute checks a scanned code and makes a verified record. A
// second scan by the same user for the same activity is not an
// error — it just returns the record already made.
func (uc *VerifyScanUseCase) Execute(ctx context.Context, in VerifyScanInput) (*domain.AttendanceVerification, error) {
	activityID, qrCodeID, found, err := uc.qrCodes.ResolveCode(ctx, in.ScannedCode)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domain.ErrInvalidQRCode
	}

	existing, err := uc.verifications.FindByActivityAndUser(ctx, activityID, in.UserID)
	if err != nil && err != domain.ErrVerificationNotFound {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	verification := &domain.AttendanceVerification{
		ID:         uuid.New(),
		ActivityID: activityID,
		UserID:     in.UserID,
		QRCodeID:   qrCodeID,
		VerifiedAt: time.Now(),
	}
	if err := uc.verifications.Create(ctx, verification); err != nil {
		return nil, err
	}

	// Best-effort, same pattern as hangout's archiveCreator call in
	// cancel_hangout.go: the verification is already saved, so a
	// badge-awarding failure is logged, not returned to the caller.
	if err := uc.badges.AwardIfBadgeExists(ctx, in.UserID, activityID); err != nil {
		log.Printf("attendance: failed to award badge for user %s, activity %s: %v", in.UserID, activityID, err)
	}

	return verification, nil
}
