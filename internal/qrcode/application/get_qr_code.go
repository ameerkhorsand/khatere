package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/qrcode/domain"
	"github.com/google/uuid"
)

type GetQRCodeUseCase struct {
	qrCodes    domain.QRCodeRepository
	activities ActivityOwnershipChecker
}

func NewGetQRCodeUseCase(qrCodes domain.QRCodeRepository, activities ActivityOwnershipChecker) *GetQRCodeUseCase {
	return &GetQRCodeUseCase{qrCodes: qrCodes, activities: activities}
}

type GetQRCodeInput struct {
	ActivityID uuid.UUID
	HostID     uuid.UUID
}

// Execute returns the host's own QR code for the host dashboard
// screen. Only the owning host may fetch it — a plain user never
// needs the raw code, only the scan endpoint (attendance domain).
func (uc *GetQRCodeUseCase) Execute(ctx context.Context, in GetQRCodeInput) (*domain.QRCode, error) {
	owned, err := uc.activities.IsOwnedByHost(ctx, in.ActivityID, in.HostID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrNotHostActivity
	}
	return uc.qrCodes.FindByActivityID(ctx, in.ActivityID)
}
