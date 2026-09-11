package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// Safety-net TTLs. Detail can sit a little longer than list, since
// a stale single-activity read is lower-impact than a stale feed.
const (
	detailTTL = 10 * time.Minute
	listTTL   = 2 * time.Minute
)

const listKey = "activity:list:approved"

type ActivityCache struct {
	client *goredis.Client
}

func NewActivityCache(client *goredis.Client) *ActivityCache {
	return &ActivityCache{client: client}
}

func detailKey(activityID uuid.UUID) string {
	return "activity:detail:" + activityID.String()
}

func (c *ActivityCache) GetDetail(ctx context.Context, activityID uuid.UUID) (*domain.Activity, bool, error) {
	raw, err := c.client.Get(ctx, detailKey(activityID)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var activity domain.Activity
	if err := json.Unmarshal(raw, &activity); err != nil {
		return nil, false, err
	}
	return &activity, true, nil
}

func (c *ActivityCache) SetDetail(ctx context.Context, activity *domain.Activity) error {
	raw, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, detailKey(activity.ID), raw, detailTTL).Err()
}

func (c *ActivityCache) InvalidateDetail(ctx context.Context, activityID uuid.UUID) error {
	return c.client.Del(ctx, detailKey(activityID)).Err()
}

func (c *ActivityCache) GetList(ctx context.Context) ([]domain.Activity, bool, error) {
	raw, err := c.client.Get(ctx, listKey).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var activities []domain.Activity
	if err := json.Unmarshal(raw, &activities); err != nil {
		return nil, false, err
	}
	return activities, true, nil
}

func (c *ActivityCache) SetList(ctx context.Context, activities []domain.Activity) error {
	raw, err := json.Marshal(activities)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, listKey, raw, listTTL).Err()
}

func (c *ActivityCache) InvalidateList(ctx context.Context) error {
	return c.client.Del(ctx, listKey).Err()
}
