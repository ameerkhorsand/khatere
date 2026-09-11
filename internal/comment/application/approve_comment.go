package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

type ApproveCommentUseCase struct {
	comments   domain.CommentRepository
	summarizer domain.SummaryRegenerator
	notify     NotificationPublisher
}

func NewApproveCommentUseCase(comments domain.CommentRepository, summarizer domain.SummaryRegenerator, notify NotificationPublisher) *ApproveCommentUseCase {
	return &ApproveCommentUseCase{comments: comments, summarizer: summarizer, notify: notify}
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

	metadata, _ := json.Marshal(map[string]string{
		"comment_id":  comment.ID.String(),
		"activity_id": comment.ActivityID.String(),
	})
	if err := uc.notify.Publish(ctx, notificationdomain.Event{
		Type:        notificationdomain.TypeCommentApproved,
		RecipientID: comment.UserID,
		ActorID:     in.ModeratorID,
		Metadata:    metadata,
	}); err != nil {
		log.Printf("comment: failed to publish approval notification for %s: %v", comment.ID, err)
	}

	return comment, nil
}
