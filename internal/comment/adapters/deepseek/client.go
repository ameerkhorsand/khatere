// Package deepseek is the one place in the codebase that knows
// DeepSeek's API shape. Nothing outside this package imports
// net/http for this feature, and nothing outside this package
// knows the request/response JSON shape below — the rest of the
// app only ever talks to domain.CommentSummarizer.
package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.deepseek.com"

// Client calls DeepSeek's OpenAI-compatible chat completions
// endpoint. It satisfies internal/comment/domain.CommentSummarizer.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient builds a DeepSeek client. apiKey must be non-empty —
// callers that don't have a key should wire up Disabled (see
// noop.go) instead of calling this.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		model:   "deepseek-chat",
		httpClient: &http.Client{
			// A summary is a background/best-effort side effect
			// (see ApproveCommentUseCase) — this timeout bounds how
			// long a moderator's approve request can be held up by
			// a slow vendor call, without needing the caller to
			// know anything about it.
			Timeout: 20 * time.Second,
		},
	}
}

// maxComments and maxBodyRunes are a second, defensive cap on
// prompt size — the use case in Step 4 already caps what it sends,
// but the adapter doesn't assume that stays true forever.
const (
	maxComments  = 40
	maxBodyRunes = 400
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Summarize(ctx context.Context, commentBodies []string) (string, error) {
	prompt := buildPrompt(commentBodies)

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		// Low temperature: this is a factual roll-up of what people
		// said, not creative writing — it should read the same way
		// twice given the same comments.
		Temperature: 0.3,
		MaxTokens:   300,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("deepseek: encoding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("deepseek: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("deepseek: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("deepseek: reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deepseek: unexpected status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("deepseek: decoding response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("deepseek: response had no choices")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

const systemPrompt = "You summarize user comments left on an activity in a hangout-planning app. " +
	"Write a short, neutral summary (2-4 sentences) of the common themes, overall sentiment, " +
	"and any recurring praise or complaints. Do not quote any comment directly. " +
	"Do not invent details that are not present in the comments."

func buildPrompt(commentBodies []string) string {
	if len(commentBodies) > maxComments {
		commentBodies = commentBodies[:maxComments]
	}

	var b strings.Builder
	b.WriteString("Comments:\n")
	for i, body := range commentBodies {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncate(body, maxBodyRunes)))
	}
	return b.String()
}

func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}
