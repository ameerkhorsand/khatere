package config

import (
	"log"
	"os"
	"time"
)

// Config holds all settings the app reads from the environment.
// Load this once in main.go. Do not call os.Getenv anywhere else.
type Config struct {
	DatabaseURL        string
	JWTSecret          string
	RedisAddr          string
	MinIOEndpoint      string
	MinIOPublicURL     string
	MinIORootUser      string
	MinIORootPassword  string
	ArchiveMediaBucket string
	Port               string
	// DeepSeekAPIKey is optional (envOrDefault, not mustEnv): AI
	// comment summaries are a nice-to-have riding on top of the
	// Comment domain, not a feature the whole server should refuse
	// to start over. See internal/comment/adapters/deepseek.
	DeepSeekAPIKey string
	// GapGPTAPIKey/GapGPTModel: an alternative AI summarizer, routed
	// through gapGPT's OpenAI-compatible proxy. Also optional, same
	// reasoning as DeepSeekAPIKey. See internal/comment/adapters/gapgpt.
	GapGPTAPIKey string
	GapGPTModel  string
	// RecommendationRefreshInterval/RecommendationStaleAfter drive
	// the background cache-refresh worker (Step 5): how often it
	// wakes up, and how old a cached user's suggestions must be
	// before that user is refreshed. See
	// internal/recommendation/adapters/worker.
	RecommendationRefreshInterval time.Duration
	RecommendationStaleAfter      time.Duration
}

// Load reads all config values from the environment.
// It stops the program if a required value is missing.
func Load() *Config {
	return &Config{
		DatabaseURL:        mustEnv("DATABASE_URL"),
		JWTSecret:          mustEnv("JWT_SECRET"),
		RedisAddr:          mustEnv("REDIS_ADDR"),
		MinIOEndpoint:      mustEnv("MINIO_ENDPOINT"),
		MinIOPublicURL:     envOrDefault("MINIO_PUBLIC_URL", "http://localhost:9000"),
		MinIORootUser:      mustEnv("MINIO_ROOT_USER"),
		MinIORootPassword:  mustEnv("MINIO_ROOT_PASSWORD"),
		ArchiveMediaBucket: envOrDefault("ARCHIVE_MEDIA_BUCKET", "archive-media"),
		Port:               envOrDefault("PORT", "8080"),
		DeepSeekAPIKey:     envOrDefault("DEEPSEEK_API_KEY", ""),
		GapGPTAPIKey:       envOrDefault("GAPGPT_API_KEY", ""),
		GapGPTModel:        envOrDefault("GAPGPT_MODEL", "gpt-4o-mini"),

		RecommendationRefreshInterval: envDurationOrDefault("RECOMMENDATION_REFRESH_INTERVAL", 10*time.Minute),
		RecommendationStaleAfter:      envDurationOrDefault("RECOMMENDATION_STALE_AFTER", 30*time.Minute),
	}
}

// mustEnv reads a required env var. It stops the program if the var is empty.
func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return val
}

// envOrDefault reads an optional env var. It returns fallback if the var is empty.
func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// envDurationOrDefault reads an optional env var as a Go duration
// string (e.g. "10m", "30s"). It returns fallback if the var is
// empty or fails to parse — a bad duration string should not stop
// the whole server from starting, just fall back quietly.
func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("config: invalid duration for %s (%q), using default %s", key, val, fallback)
		return fallback
	}
	return d
}
