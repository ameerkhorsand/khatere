package deepseek

import (
	"context"
	"errors"
)

// ErrDisabled means DEEPSEEK_API_KEY was never set. The comment
// summary endpoint (Step 6) checks for this specific error and
// turns it into a clear, deliberate response, instead of a generic
// 500 that looks like a bug.
var ErrDisabled = errors.New("AI comment summaries are disabled: DEEPSEEK_API_KEY not set")

// Disabled satisfies domain.CommentSummarizer without ever making a
// network call. server.go wires this in instead of Client when the
// API key is empty, so a missing key degrades one feature instead
// of stopping the whole server from starting (see config.go: this
// key is optional, not mustEnv).
type Disabled struct{}

func (Disabled) Summarize(ctx context.Context, commentBodies []string) (string, error) {
	return "", ErrDisabled
}
