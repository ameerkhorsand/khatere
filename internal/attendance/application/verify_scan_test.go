package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/attendance/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeQRCodeLookup struct {
	activityID uuid.UUID
	qrCodeID   uuid.UUID
	found      bool
	err        error
}

func (f *fakeQRCodeLookup) ResolveCode(ctx context.Context, code string) (uuid.UUID, uuid.UUID, bool, error) {
	return f.activityID, f.qrCodeID, f.found, f.err
}

type fakeBadgeAwarder struct {
	err         error
	awardCalled bool
}

func (f *fakeBadgeAwarder) AwardIfBadgeExists(ctx context.Context, userID, activityID uuid.UUID) error {
	f.awardCalled = true
	return f.err
}

type fakeVerificationRepo struct {
	existing  *domain.AttendanceVerification
	findErr   error
	createErr error
	created   *domain.AttendanceVerification
}

func (f *fakeVerificationRepo) Create(ctx context.Context, v *domain.AttendanceVerification) error {
	f.created = v
	return f.createErr
}
func (f *fakeVerificationRepo) FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*domain.AttendanceVerification, error) {
	return f.existing, f.findErr
}

// --- tests ---------------------------------------------------------

func TestVerifyScanUseCase_Execute(t *testing.T) {
	t.Run("unrecognized code is rejected", func(t *testing.T) {
		qr := &fakeQRCodeLookup{found: false}
		uc := NewVerifyScanUseCase(&fakeVerificationRepo{findErr: domain.ErrVerificationNotFound}, qr, &fakeBadgeAwarder{})

		_, err := uc.Execute(context.Background(), VerifyScanInput{ScannedCode: "bogus"})

		if !errors.Is(err, domain.ErrInvalidQRCode) {
			t.Errorf("got %v, want ErrInvalidQRCode", err)
		}
	})

	t.Run("second scan returns the existing verification, does not create or award again", func(t *testing.T) {
		existing := &domain.AttendanceVerification{ID: uuid.New()}
		repo := &fakeVerificationRepo{existing: existing}
		badges := &fakeBadgeAwarder{}
		qr := &fakeQRCodeLookup{found: true, activityID: uuid.New(), qrCodeID: uuid.New()}
		uc := NewVerifyScanUseCase(repo, qr, badges)

		got, err := uc.Execute(context.Background(), VerifyScanInput{ScannedCode: "valid"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != existing.ID {
			t.Errorf("expected the existing verification to be returned unchanged")
		}
		if repo.created != nil {
			t.Errorf("expected no new verification to be created on a repeat scan")
		}
		if badges.awardCalled {
			t.Errorf("expected no badge award attempt on a repeat scan")
		}
	})

	t.Run("badge award failure does not fail the scan", func(t *testing.T) {
		repo := &fakeVerificationRepo{findErr: domain.ErrVerificationNotFound}
		badges := &fakeBadgeAwarder{err: errors.New("badge service down")}
		qr := &fakeQRCodeLookup{found: true, activityID: uuid.New(), qrCodeID: uuid.New()}
		uc := NewVerifyScanUseCase(repo, qr, badges)

		got, err := uc.Execute(context.Background(), VerifyScanInput{UserID: uuid.New(), ScannedCode: "valid"})

		if err != nil {
			t.Fatalf("scan must succeed even if badge awarding fails, got: %v", err)
		}
		if got == nil {
			t.Fatalf("expected a verification to be returned")
		}
	})

	t.Run("first scan creates a verification and attempts a badge award", func(t *testing.T) {
		repo := &fakeVerificationRepo{findErr: domain.ErrVerificationNotFound}
		badges := &fakeBadgeAwarder{}
		activityID, qrCodeID, userID := uuid.New(), uuid.New(), uuid.New()
		qr := &fakeQRCodeLookup{found: true, activityID: activityID, qrCodeID: qrCodeID}
		uc := NewVerifyScanUseCase(repo, qr, badges)

		got, err := uc.Execute(context.Background(), VerifyScanInput{UserID: userID, ScannedCode: "valid"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ActivityID != activityID || got.QRCodeID != qrCodeID || got.UserID != userID {
			t.Errorf("verification fields not set from the resolved code / input")
		}
		if repo.created == nil {
			t.Errorf("expected a new verification to be created")
		}
		if !badges.awardCalled {
			t.Errorf("expected a badge award attempt")
		}
	})
}
