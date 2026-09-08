package main

import (
	"context"
	"net/http"
	"time"

	authHTTP "github.com/bLorax/khatere-backend/internal/auth/adapters/http"
	authPG "github.com/bLorax/khatere-backend/internal/auth/adapters/postgres"
	authApp "github.com/bLorax/khatere-backend/internal/auth/application"

	hostHTTP "github.com/bLorax/khatere-backend/internal/host/adapters/http"
	hostPG "github.com/bLorax/khatere-backend/internal/host/adapters/postgres"
	hostApp "github.com/bLorax/khatere-backend/internal/host/application"

	moderatorHTTP "github.com/bLorax/khatere-backend/internal/moderator/adapters/http"
	moderatorPG "github.com/bLorax/khatere-backend/internal/moderator/adapters/postgres"
	moderatorApp "github.com/bLorax/khatere-backend/internal/moderator/application"

	userHTTP "github.com/bLorax/khatere-backend/internal/user/adapters/http"
	userPG "github.com/bLorax/khatere-backend/internal/user/adapters/postgres"
	userApp "github.com/bLorax/khatere-backend/internal/user/application"

	"github.com/bLorax/khatere-backend/internal/platform/config"
	"github.com/bLorax/khatere-backend/internal/platform/httpserver/middleware"
	miniocl "github.com/bLorax/khatere-backend/internal/platform/minio"
	rediscl "github.com/bLorax/khatere-backend/internal/platform/redis"
	"github.com/bLorax/khatere-backend/internal/platform/security"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	miniogo "github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
)

// deps bundles the already-connected infrastructure clients that the
// router needs. main.go builds this after it has pinged everything.
type deps struct {
	cfg         *config.Config
	pool        *pgxpool.Pool
	redisClient *goredis.Client
	minioClient *miniogo.Client
}

// newRouter wires every bounded context's repositories, use cases, and
// HTTP handlers, then returns a ready-to-run gin.Engine.
func newRouter(d deps) *gin.Engine {
	// --- Security ---
	hasher := security.NewBcryptHasher(0)
	tokens := security.NewJWTTokenService([]byte(d.cfg.JWTSecret), 15*time.Minute)
	refreshTTL := 30 * 24 * time.Hour

	// --- Auth domain wiring ---
	accountRepo := authPG.NewAccountRepository(d.pool)
	refreshTokenRepo := authPG.NewRefreshTokenRepository(d.pool)

	registerUC := authApp.NewRegisterUseCase(accountRepo, hasher)
	loginUC := authApp.NewLoginUseCase(accountRepo, refreshTokenRepo, hasher, tokens, refreshTTL)
	refreshUC := authApp.NewRefreshUseCase(accountRepo, refreshTokenRepo, tokens, refreshTTL)

	authHandlers := authHTTP.NewHandlers(registerUC, loginUC, refreshUC)

	// --- User domain wiring ---
	userRepo := userPG.NewUserRepository(d.pool)
	interestRepo := userPG.NewInterestRepository(d.pool)

	createProfileUC := userApp.NewCreateProfileUseCase(userRepo)
	updateProfileUC := userApp.NewUpdateProfileUseCase(userRepo)
	setInterestsUC := userApp.NewSetInterestsUseCase(interestRepo)
	listInterestsUC := userApp.NewListInterestsUseCase(interestRepo)
	listCatalogUC := userApp.NewListInterestCatalogUseCase(interestRepo)

	userHandlers := userHTTP.NewHandlers(createProfileUC, updateProfileUC, setInterestsUC, listInterestsUC, listCatalogUC)

	// --- Host domain wiring ---
	hostRepo := hostPG.NewHostRepository(d.pool)
	createHostProfileUC := hostApp.NewCreateHostProfileUseCase(hostRepo)
	hostHandlers := hostHTTP.NewHandlers(createHostProfileUC)

	// --- Moderator domain wiring ---
	moderatorRepo := moderatorPG.NewModeratorRepository(d.pool)
	createModeratorProfileUC := moderatorApp.NewCreateModeratorProfileUseCase(moderatorRepo)
	moderatorHandlers := moderatorHTTP.NewHandlers(createModeratorProfileUC)

	// --- Router ---
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		reqCtx := c.Request.Context()
		healthCheck(reqCtx, c, d)
	})

	authGroup := router.Group("/auth")
	authHandlers.RegisterRoutes(authGroup)

	authMW := middleware.AuthRequired(d.cfg.JWTSecret)

	userGroup := router.Group("/user")
	userGroup.Use(authMW)
	userHandlers.RegisterRoutes(userGroup)

	hostGroup := router.Group("/host")
	hostGroup.Use(authMW)
	hostHandlers.RegisterRoutes(hostGroup)

	moderatorGroup := router.Group("/moderator")
	moderatorGroup.Use(authMW)
	moderatorHandlers.RegisterRoutes(moderatorGroup)

	return router
}

// healthCheck pings every downstream dependency and reports the first
// failure it finds.
func healthCheck(ctx context.Context, c *gin.Context, d deps) {
	if err := d.pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "postgres: " + err.Error()})
		return
	}
	if err := rediscl.Ping(ctx, d.redisClient); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "redis: " + err.Error()})
		return
	}
	if err := miniocl.Ping(ctx, d.minioClient); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "minio: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
