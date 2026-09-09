package config

import (
	"log"
	"os"
)

// Config holds all settings the app reads from the environment.
// Load this once in main.go. Do not call os.Getenv anywhere else.
type Config struct {
	DatabaseURL        string
	JWTSecret          string
	RedisAddr          string
	MinIOEndpoint      string
	MinIORootUser      string
	MinIORootPassword  string
	ArchiveMediaBucket string
	DeepSeekAPIKey     string
	DeepSeekBaseURL    string
	DeepSeekModel      string
	Port               string
}

// Load reads all config values from the environment.
// It stops the program if a required value is missing.
func Load() *Config {
	return &Config{
		DatabaseURL:        mustEnv("DATABASE_URL"),
		JWTSecret:          mustEnv("JWT_SECRET"),
		RedisAddr:          mustEnv("REDIS_ADDR"),
		MinIOEndpoint:      mustEnv("MINIO_ENDPOINT"),
		MinIORootUser:      mustEnv("MINIO_ROOT_USER"),
		MinIORootPassword:  mustEnv("MINIO_ROOT_PASSWORD"),
		ArchiveMediaBucket: envOrDefault("ARCHIVE_MEDIA_BUCKET", "archive-media"),
		// DeepSeekAPIKey is intentionally optional (envOrDefault, not
		// mustEnv): the AI comment summary is one feature among many,
		// and a missing key should disable it, not stop the whole API
		// from starting. See internal/comment/adapters/deepseek for
		// how an empty key is handled.
		DeepSeekAPIKey:  envOrDefault("DEEPSEEK_API_KEY", ""),
		DeepSeekBaseURL: envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:   envOrDefault("DEEPSEEK_MODEL", "deepseek-chat"),
		Port:            envOrDefault("PORT", "8080"),
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
