package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

type ApproveCommentUseCase struct {
	comments   domain.CommentRepository
	summarizer domain.SummaryRegenerator
}

func NewApproveCommentUseCase(comments domain.CommentRepository, summarizer domain.SummaryRegenerator) *ApproveCommentUseCase {
	return &ApproveCommentUseCase{comments: comments, summarizer: summarizer}
}

type ApproveCommentInput struct {
	CommentID   uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *ApproveCommentUseCase) Execute(ctx context.Context, in ApproveCommentInput) (*domain.Comment, error) {
	comment, err := uc.comments.FindByID(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}

	if comment.Status != domain.CommentStatusPending {
		return nil, domain.ErrCommentNotPending
	}

	now := time.Now()
	comment.Status = domain.CommentStatusApproved
	comment.ReviewedBy = &in.ModeratorID
	comment.ReviewedAt = &now

	if err := uc.comments.Update(ctx, comment); err != nil {
		return nil, err
	}

	// Best-effort, same pattern as UpdateHangoutStatusUseCase's
	// archive creation: the approval already succeeded and is the
	// thing the moderator asked for. A summary regeneration failure
	// (e.g. DeepSeek is down, or no key configured) is logged, not
	// returned — it must never make an otherwise-successful approve
	// look like it failed.
	if err := uc.summarizer.Execute(ctx, comment.ActivityID); err != nil {
		log.Printf("comment: failed to regenerate summary for activity %s: %v", comment.ActivityID, err)
	}

	return comment, nil
}
