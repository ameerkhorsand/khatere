package domain

import (
	"context"

	"github.com/google/uuid"
)

type QRCodeRepository interface {
	// Create inserts a new QR code. Fails on the UNIQUE(activity_id)
	// constraint (migration 0018) if the activity already has one —
	// callers should check FindByActivityID first, same pattern as
	// rating's Upsert-after-FindByActivityAndUser.
	Create(ctx context.Context, qr *QRCode) error

	FindByActivityID(ctx context.Context, activityID uuid.UUID) (*QRCode, error)

	// FindByCode looks up the QR code a scan just read off the
	// image. Used by the attendance domain, not by this one.
	FindByCode(ctx context.Context, code string) (*QRCode, error)
}
