package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// ttl is a safety net, not the primary invalidation mechanism —
// CreateRatingUseCase deletes the key explicitly on every new or
// changed rating. This just bounds how stale a missed invalidation
// could ever leave a value.
const ttl = 5 * time.Minute

type SummaryCache struct {
	client *goredis.Client
}

func NewSummaryCache(client *goredis.Client) *SummaryCache {
	return &SummaryCache{client: client}
}

func key(activityID uuid.UUID) string {
	return "rating:summary:" + activityID.String()
}

func (c *SummaryCache) Get(ctx context.Context, activityID uuid.UUID) (domain.Summary, bool, error) {
	raw, err := c.client.Get(ctx, key(activityID)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return domain.Summary{}, false, nil
	}
	if err != nil {
		return domain.Summary{}, false, err
	}

	var summary domain.Summary
	if err := json.Unmarshal(raw, &summary); err != nil {
		return domain.Summary{}, false, err
	}
	return summary, true, nil
}

func (c *SummaryCache) Set(ctx context.Context, activityID uuid.UUID, summary domain.Summary) error {
	raw, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key(activityID), raw, ttl).Err()
}

func (c *SummaryCache) Invalidate(ctx context.Context, activityID uuid.UUID) error {
	return c.client.Del(ctx, key(activityID)).Err()
}
