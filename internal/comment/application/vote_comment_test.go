package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeCommentRepo struct {
	comment   *domain.Comment
	findErr   error
	updateErr error
}

func (f *fakeCommentRepo) Create(ctx context.Context, c *domain.Comment) error { return nil }
func (f *fakeCommentRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	return f.comment, f.findErr
}
func (f *fakeCommentRepo) ListApprovedByActivity(ctx context.Context, activityID uuid.UUID) ([]domain.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) ListPendingQueue(ctx context.Context) ([]domain.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) Update(ctx context.Context, c *domain.Comment) error { return f.updateErr }

type fakeCommentVoteRepo struct {
	upsertErr error
	lastVote  *domain.CommentVote // captured so a test can assert on it
}

func (f *fakeCommentVoteRepo) Upsert(ctx context.Context, v *domain.CommentVote) error {
	f.lastVote = v
	return f.upsertErr
}
func (f *fakeCommentVoteRepo) FindVote(ctx context.Context, commentID, userID uuid.UUID) (*domain.CommentVote, error) {
	return nil, nil
}
func (f *fakeCommentVoteRepo) TallyForComments(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.VoteTally, error) {
	return nil, nil
}

// --- tests ---------------------------------------------------------

func TestVoteCommentUseCase_Execute(t *testing.T) {
	approvedComment := &domain.Comment{ID: uuid.New(), Status: domain.CommentStatusApproved}
	pendingComment := &domain.Comment{ID: uuid.New(), Status: domain.CommentStatusPending}

	tests := []struct {
		name        string
		commentRepo *fakeCommentRepo
		voteRepo    *fakeCommentVoteRepo
		input       VoteCommentInput
		wantErr     error
	}{
		{
			name:        "invalid vote value is rejected",
			commentRepo: &fakeCommentRepo{},
			voteRepo:    &fakeCommentVoteRepo{},
			input:       VoteCommentInput{Value: 99},
			wantErr:     domain.ErrInvalidVoteValue,
		},
		{
			name:        "lookup failure is passed through",
			commentRepo: &fakeCommentRepo{findErr: errors.New("db down")},
			voteRepo:    &fakeCommentVoteRepo{},
			input:       VoteCommentInput{Value: domain.VoteUp},
			wantErr:     errors.New("db down"),
		},
		{
			name:        "pending comment cannot be voted on",
			commentRepo: &fakeCommentRepo{comment: pendingComment},
			voteRepo:    &fakeCommentVoteRepo{},
			input:       VoteCommentInput{CommentID: pendingComment.ID, Value: domain.VoteUp},
			wantErr:     domain.ErrCommentNotFound,
		},
		{
			name:        "success upserts the vote",
			commentRepo: &fakeCommentRepo{comment: approvedComment},
			voteRepo:    &fakeCommentVoteRepo{},
			input:       VoteCommentInput{CommentID: approvedComment.ID, UserID: uuid.New(), Value: domain.VoteDown},
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewVoteCommentUseCase(tt.commentRepo, tt.voteRepo)

			err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.voteRepo.lastVote == nil {
					t.Fatalf("expected a vote to be upserted")
				}
				if tt.voteRepo.lastVote.Value != tt.input.Value {
					t.Errorf("upserted vote value = %v, want %v", tt.voteRepo.lastVote.Value, tt.input.Value)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("got error %q, want %q", err, tt.wantErr)
			}
		})
	}
}
