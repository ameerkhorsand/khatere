package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------
// CommentSummary
// -----------------------------------------------------------------

// CommentSummary is the cached AI summary for one activity's
// approved comments. GeneratedAt is nil until the first successful
// generation — that's how a caller tells "never generated" apart
// from "generated, but there were zero comments at the time".
type CommentSummary struct {
	ActivityID   uuid.UUID
	Summary      string
	CommentCount int
	GeneratedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var ErrSummaryNotFound = errors.New("no comment summary generated yet")

// CommentSummaryRepository stores and reads the one cached row per
// activity described above.
type CommentSummaryRepository interface {
	// Upsert writes the summary for an activity, creating the row
	// on first write and overwriting it on every regeneration after
	// that — there is only ever one row per activity.
	Upsert(ctx context.Context, summary *CommentSummary) error

	// FindByActivityID returns ErrSummaryNotFound if no summary has
	// been generated for this activity yet.
	FindByActivityID(ctx context.Context, activityID uuid.UUID) (*CommentSummary, error)
}

// -----------------------------------------------------------------
// CommentSummarizer — the AI vendor port
// -----------------------------------------------------------------

// CommentSummarizer turns a set of approved comment bodies into a
// short written summary. This interface is deliberately vendor-free
// — it says nothing about DeepSeek, HTTP, or API keys — so the
// application layer, and everything above it, never needs to change
// if the AI vendor changes. Same small-port pattern as
// AttendanceChecker in the rating domain and ArchiveCreator in the
// hangout domain: a use case depends on this interface, and a
// separate adapter package satisfies it.
type CommentSummarizer interface {
	Summarize(ctx context.Context, commentBodies []string) (string, error)
}

// -----------------------------------------------------------------
// SummaryRegenerator — the cross-use-case port
// -----------------------------------------------------------------

// SummaryRegenerator lets ApproveCommentUseCase trigger a summary
// refresh without depending on RegenerateCommentSummaryUseCase's
// constructor arguments or how many other things it depends on.
// ApproveCommentUseCase only needs to know "regeneration is a thing
// I can ask for", not how it happens.
//
// Unlike ArchiveCreator (hangout domain calling into the archive
// domain, a real cross-boundary call needing a translation shim),
// this port and its implementation both live inside the comment
// domain. So the method here is named Execute, matching the house
// convention every other use case follows — RegenerateCommentSummaryUseCase
// satisfies this interface directly, with no adapter in between.
type SummaryRegenerator interface {
	Execute(ctx context.Context, activityID uuid.UUID) error
}
