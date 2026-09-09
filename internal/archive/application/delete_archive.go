package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

// DeleteArchiveUseCase composes MarkArchiveDeletedUseCase and
// CheckAndPurgeArchiveUseCase into one call, run inside a single
// database transaction. Without the transaction, two concurrent
// delete requests from the last two participants could both read
// "not unanimous yet" a moment before either write landed.
//
// Note: CheckAndPurgeArchiveUseCase also calls out to MediaStorage
// (MinIO) inside this transaction. A MinIO delete cannot be rolled
// back if the surrounding DB transaction later fails to commit, so
// in the rare case of a commit failure after a successful file
// delete, the file and the database can briefly disagree. Acceptable
// for this phase; revisit with an outbox pattern if it matters later.
type DeleteArchiveUseCase struct {
	tx    domain.Transactor
	mark  *MarkArchiveDeletedUseCase
	purge *CheckAndPurgeArchiveUseCase
}

func NewDeleteArchiveUseCase(tx domain.Transactor, mark *MarkArchiveDeletedUseCase, purge *CheckAndPurgeArchiveUseCase) *DeleteArchiveUseCase {
	return &DeleteArchiveUseCase{tx: tx, mark: mark, purge: purge}
}

type DeleteArchiveInput struct {
	ArchiveID uuid.UUID
	UserID    uuid.UUID
}

// Execute returns whether this call was the one that purged the
// archive (i.e. this user was the last participant to ask).
func (uc *DeleteArchiveUseCase) Execute(ctx context.Context, in DeleteArchiveInput) (bool, error) {
	var purged bool

	err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.mark.Execute(ctx, MarkArchiveDeletedInput{
			ArchiveID: in.ArchiveID,
			UserID:    in.UserID,
		}); err != nil {
			return err
		}

		didPurge, err := uc.purge.Execute(ctx, CheckAndPurgeArchiveInput{ArchiveID: in.ArchiveID})
		if err != nil {
			return err
		}
		purged = didPurge
		return nil
	})

	return purged, err
}
