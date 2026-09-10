package application

import (
	"context"
	"fmt"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

// RegenerateCommentSummaryUseCase rebuilds the cached AI summary
// for one activity from its currently-approved comments. It is
// called from ApproveCommentUseCase (via the SummaryRegenerator
// port) every time a comment gets approved — that's the only event
// that can add a new comment to what the summary should cover.
type RegenerateCommentSummaryUseCase struct {
	comments   domain.CommentRepository
	summaries  domain.CommentSummaryRepository
	summarizer domain.CommentSummarizer
}

func NewRegenerateCommentSummaryUseCase(
	comments domain.CommentRepository,
	summaries domain.CommentSummaryRepository,
	summarizer domain.CommentSummarizer,
) *RegenerateCommentSummaryUseCase {
	return &RegenerateCommentSummaryUseCase{comments: comments, summaries: summaries, summarizer: summarizer}
}

func (uc *RegenerateCommentSummaryUseCase) Execute(ctx context.Context, activityID uuid.UUID) error {
	comments, err := uc.comments.ListApprovedByActivity(ctx, activityID)
	if err != nil {
		return fmt.Errorf("listing approved comments: %w", err)
	}

	bodies := make([]string, len(comments))
	for i, c := range comments {
		bodies[i] = c.Body
	}

	// Zero approved comments is not an error — it's a valid state
	// (a brand-new activity, or one where nothing's been approved
	// yet). Store an empty summary rather than calling the AI
	// vendor with nothing to summarize.
	var summaryText string
	if len(bodies) > 0 {
		summaryText, err = uc.summarizer.Summarize(ctx, bodies)
		if err != nil {
			return fmt.Errorf("summarizing comments: %w", err)
		}
	}

	now := time.Now()
	return uc.summaries.Upsert(ctx, &domain.CommentSummary{
		ActivityID:   activityID,
		Summary:      summaryText,
		CommentCount: len(bodies),
		GeneratedAt:  &now,
	})
}
