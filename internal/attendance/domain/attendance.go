package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// AttendanceVerification is one row per user per activity scan. A
// user makes this by scanning the activity's fixed QR code
// (qrcode domain) while at the host's activity.
type AttendanceVerification struct {
	ID         uuid.UUID
	ActivityID uuid.UUID
	UserID     uuid.UUID
	QRCodeID   uuid.UUID
	VerifiedAt time.Time
}

var (
	ErrVerificationNotFound = errors.New("attendance verification not found")
	// ErrInvalidQRCode covers both a code that never existed and one
	// that's just been mistyped/misscanned — the scan endpoint does
	// not distinguish the two, so a user can't probe for real codes.
	ErrInvalidQRCode = errors.New("this qr code is not valid")
)
