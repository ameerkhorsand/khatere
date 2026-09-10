package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// QRCode is the one fixed code for a host activity. Code is the
// opaque string encoded into the QR image shown by the frontend —
// this backend never renders an image, only the string.
type QRCode struct {
	ID         uuid.UUID
	ActivityID uuid.UUID
	Code       string
	CreatedAt  time.Time
}

var (
	ErrQRCodeNotFound  = errors.New("qr code not found")
	ErrNotHostActivity = errors.New("only the host who created this activity may manage its qr code")
)
