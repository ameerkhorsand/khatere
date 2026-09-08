package main

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/platform/config"
	miniocl "github.com/bLorax/khatere-backend/internal/platform/minio"
	rediscl "github.com/bLorax/khatere-backend/internal/platform/redis"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	// --- Database ---
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping failed: %v", err)
	}

	// --- Redis ---
	redisClient := rediscl.NewClient(cfg.RedisAddr)
	defer redisClient.Close()

	if err := rediscl.Ping(ctx, redisClient); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	// --- MinIO ---
	minioClient, err := miniocl.NewClient(cfg.MinIOEndpoint, cfg.MinIORootUser, cfg.MinIORootPassword)
	if err != nil {
		log.Fatalf("failed to build minio client: %v", err)
	}

	if err := miniocl.Ping(ctx, minioClient); err != nil {
		log.Fatalf("minio ping failed: %v", err)
	}

	// --- Router ---
	router := newRouter(deps{
		cfg:         cfg,
		pool:        pool,
		redisClient: redisClient,
		minioClient: minioClient,
	})

	// --- Start server ---
	log.Printf("starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
