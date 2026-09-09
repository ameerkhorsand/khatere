// Package archivecreator is the one place the hangout module reaches
// across into the archive module — the mirror of how
// internal/archive/adapters/hangout reaches the other way.
package archivecreator

import (
	"context"
	"errors"

	archiveApp "github.com/bLorax/khatere-backend/internal/archive/application"
	archivedomain "github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/google/uuid"
)

type Creator struct {
	uc *archiveApp.CreateArchiveUseCase
}

func New(uc *archiveApp.CreateArchiveUseCase) *Creator {
	return &Creator{uc: uc}
}

func (c *Creator) CreateArchive(ctx context.Context, hangoutID uuid.UUID) error {
	_, err := c.uc.Execute(ctx, archiveApp.CreateArchiveInput{HangoutID: hangoutID})
	if err != nil && errors.Is(err, archivedomain.ErrArchiveAlreadyExists) {
		return nil // idempotent — already created, not an error
	}
	return err
}
