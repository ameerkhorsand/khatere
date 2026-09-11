package application

import (
	"context"
	"io"
	"log"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type UploadProfilePictureUseCase struct {
	users   domain.UserRepository
	storage domain.ProfileStorage
}

func NewUploadProfilePictureUseCase(users domain.UserRepository, storage domain.ProfileStorage) *UploadProfilePictureUseCase {
	return &UploadProfilePictureUseCase{users: users, storage: storage}
}

type UploadProfilePictureInput struct {
	AccountID   uuid.UUID
	Filename    string
	ContentType string
	Content     io.Reader
	SizeBytes   int64
}

// Execute validates the file, uploads it, and saves the new key on
// the user's profile. If the user already had a picture, the old
// object is deleted afterward — best-effort, same "fail open"
// reasoning already used for cache invalidation elsewhere in this
// project: a failed cleanup must never undo a successful upload.
func (uc *UploadProfilePictureUseCase) Execute(ctx context.Context, in UploadProfilePictureInput) (*domain.User, error) {
	if !domain.ValidContentType(in.ContentType) {
		return nil, domain.ErrInvalidProfilePictureType
	}
	if in.SizeBytes > domain.MaxProfilePictureSizeBytes {
		return nil, domain.ErrProfilePictureTooLarge
	}

	user, err := uc.users.FindByAccountID(ctx, in.AccountID)
	if err != nil {
		return nil, err
	}

	newKey, err := uc.storage.Upload(ctx, in.AccountID, in.Filename, in.ContentType, in.Content, in.SizeBytes)
	if err != nil {
		return nil, err
	}

	oldKey := user.ProfilePictureKey
	user.ProfilePictureKey = &newKey

	if err := uc.users.Update(ctx, user); err != nil {
		return nil, err
	}

	if oldKey != nil {
		if err := uc.storage.Delete(ctx, *oldKey); err != nil {
			log.Printf("user: failed to delete old profile picture %s: %v", *oldKey, err)
		}
	}

	return user, nil
}
