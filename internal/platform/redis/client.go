package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

// NewClient builds a Redis client for the given address (e.g. "redis:6379").
// It does not connect yet. Call Ping to check the connection.
func NewClient(addr string) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr: addr,
	})
}

// Ping checks that Redis answers.
func Ping(ctx context.Context, client *goredis.Client) error {
	return client.Ping(ctx).Err()
}
