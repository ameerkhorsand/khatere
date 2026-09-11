package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// KeyByIP builds a rate limit key from the caller's IP address.
// Use this for routes with no login yet, such as /auth.
func KeyByIP(prefix string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		return prefix + ":" + c.ClientIP()
	}
}

// KeyByAccountID builds a rate limit key from the logged-in account.
// Use this for routes behind AuthRequired, where "account_id" is
// already set in the Gin context.
func KeyByAccountID(prefix string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		accountID, _ := c.Get("account_id")
		return prefix + ":" + accountID.(string)
	}
}

// RateLimit blocks a caller once they cross "limit" requests within
// "window". It uses a fixed window, counted in Redis with INCR and
// EXPIRE. On a Redis error, it lets the request through (fail open),
// so a Redis outage does not take down the whole API.
func RateLimit(client *goredis.Client, keyFunc func(*gin.Context) string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		key := "ratelimit:" + keyFunc(c)

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			client.Expire(ctx, key, window)
		}
		if count > int64(limit) {
			c.Header("Retry-After", window.String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
