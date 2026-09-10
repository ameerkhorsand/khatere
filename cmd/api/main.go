package main

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/platform/config"
	miniocl "github.com/bLorax/khatere-backend/internal/platform/minio"
	rediscl "github.com/bLorax/khatere-backend/internal/platform/redis"

	"github.com/jackc/pgx/v5/pgxpool"
	miniogo "github.com/minio/minio-go/v7"
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

	// Ensure the archive media bucket exists — MinIO does not create
	// buckets on first PutObject the way S3 sometimes appears to.
	exists, err := minioClient.BucketExists(ctx, cfg.ArchiveMediaBucket)
	if err != nil {
		log.Fatalf("failed to check archive media bucket: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.ArchiveMediaBucket, miniogo.MakeBucketOptions{}); err != nil {
			log.Fatalf("failed to create archive media bucket: %v", err)
		}
		log.Printf("created minio bucket: %s", cfg.ArchiveMediaBucket)
	}

	// --- Router ---
	router, recommendationWorker := newRouter(deps{
		cfg:         cfg,
		pool:        pool,
		redisClient: redisClient,
		minioClient: minioClient,
	})

	// --- Background workers ---
	// workerCtx is cancelled on the way out of main so the worker's
	// loop stops cleanly. Note: this only runs if router.Run below
	// returns normally — a log.Fatalf elsewhere in this file exits
	// the process immediately and skips deferred cleanup, same as
	// every other defer in this function.
	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	go recommendationWorker.Start(workerCtx)

	// --- Start server ---
	log.Printf("starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
