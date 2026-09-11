package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeUserRepo struct {
	user      *domain.User
	findErr   error
	updateErr error
	updated   *domain.User
}

func (f *fakeUserRepo) Create(ctx context.Context, user *domain.User) error {
	return errors.New("not implemented")
}

func (f *fakeUserRepo) FindByAccountID(ctx context.Context, accountID uuid.UUID) (*domain.User, error) {
	return f.user, f.findErr
}

func (f *fakeUserRepo) FindByHandle(ctx context.Context, handle string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeUserRepo) Update(ctx context.Context, user *domain.User) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = user
	return nil
}

type fakeProfileStorage struct {
	uploadKey    string
	uploadErr    error
	deleteErr    error
	uploadCalled bool
	deletedKey   string
	deleteCalled bool
}

func (f *fakeProfileStorage) Upload(ctx context.Context, accountID uuid.UUID, filename string, contentType string, content io.Reader, sizeBytes int64) (string, error) {
	f.uploadCalled = true
	if f.uploadErr != nil {
		return "", f.uploadErr
	}
	return f.uploadKey, nil
}

func (f *fakeProfileStorage) Delete(ctx context.Context, storageKey string) error {
	f.deleteCalled = true
	f.deletedKey = storageKey
	return f.deleteErr
}

func (f *fakeProfileStorage) PublicURL(ctx context.Context, storageKey string) (string, error) {
	return "https://example.com/" + storageKey, nil
}

// --- tests ---------------------------------------------------------

func TestUploadProfilePictureUseCase_Execute(t *testing.T) {
	t.Run("bad content type is rejected", func(t *testing.T) {
		repo := &fakeUserRepo{user: &domain.User{AccountID: uuid.New()}}
		storage := &fakeProfileStorage{uploadKey: "users/x/new.gif"}
		uc := NewUploadProfilePictureUseCase(repo, storage)

		_, err := uc.Execute(context.Background(), UploadProfilePictureInput{
			AccountID:   uuid.New(),
			Filename:    "avatar.gif",
			ContentType: "image/gif",
			Content:     bytes.NewReader([]byte("fake")),
			SizeBytes:   4,
		})

		if !errors.Is(err, domain.ErrInvalidProfilePictureType) {
			t.Errorf("got %v, want ErrInvalidProfilePictureType", err)
		}
		if storage.uploadCalled {
			t.Errorf("expected no upload attempt for a rejected content type")
		}
	})
}
