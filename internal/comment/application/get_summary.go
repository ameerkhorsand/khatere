package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

// GetCommentSummaryUseCase is a plain read — it never talks to the
// AI vendor. It only returns whatever RegenerateCommentSummaryUseCase
// last stored. This keeps every GET request instant and free, no
// matter how the write side (or its cost) changes later.
type GetCommentSummaryUseCase struct {
	summaries domain.CommentSummaryRepository
}

func NewGetCommentSummaryUseCase(summaries domain.CommentSummaryRepository) *GetCommentSummaryUseCase {
	return &GetCommentSummaryUseCase{summaries: summaries}
}

// Execute returns domain.ErrSummaryNotFound if this activity has
// never had a comment approved (so RegenerateCommentSummaryUseCase
// has never run for it) — the HTTP layer turns that into a
// deliberate "not generated yet" response, not a 500.
func (uc *GetCommentSummaryUseCase) Execute(ctx context.Context, activityID uuid.UUID) (*domain.CommentSummary, error) {
	return uc.summaries.FindByActivityID(ctx, activityID)
}
