package main

import (
	"context"
	_ "embed"
	"net/http"
	"strings"
	"time"

	activityHTTP "github.com/bLorax/khatere-backend/internal/activity/adapters/http"
	activityPG "github.com/bLorax/khatere-backend/internal/activity/adapters/postgres"
	activityApp "github.com/bLorax/khatere-backend/internal/activity/application"

	hangoutHTTP "github.com/bLorax/khatere-backend/internal/hangout/adapters/http"
	hangoutNotif "github.com/bLorax/khatere-backend/internal/hangout/adapters/notification"
	hangoutPG "github.com/bLorax/khatere-backend/internal/hangout/adapters/postgres"
	hangoutApp "github.com/bLorax/khatere-backend/internal/hangout/application"

	circleHTTP "github.com/bLorax/khatere-backend/internal/circle/adapters/http"
	circlePG "github.com/bLorax/khatere-backend/internal/circle/adapters/postgres"
	circleApp "github.com/bLorax/khatere-backend/internal/circle/application"

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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	miniogo "github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
)

//go:embed web/khatere-api-console.html
var apiTesterHTML []byte

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

	// --- Circle domain wiring ---
	connectionRepo := circlePG.NewConnectionRepository(d.pool)
	blockRepo := circlePG.NewBlockRepository(d.pool)

	sendConnectionRequestUC := circleApp.NewSendConnectionRequestUseCase(
		connectionRepo,
		blockRepo,
	)
	acceptConnectionRequestUC := circleApp.NewAcceptConnectionRequestUseCase(
		connectionRepo,
		blockRepo,
	)
	declineConnectionRequestUC := circleApp.NewDeclineConnectionRequestUseCase(
		connectionRepo,
	)
	severConnectionUC := circleApp.NewSeverConnectionUseCase(
		connectionRepo,
	)
	blockUserUC := circleApp.NewBlockUserUseCase(
		blockRepo,
	)
	unblockUserUC := circleApp.NewUnblockUserUseCase(
		blockRepo,
	)
	listCircleUC := circleApp.NewListCircleUseCase(
		connectionRepo,
	)
	listPendingRequestsUC := circleApp.NewListPendingRequestsUseCase(
		connectionRepo,
	)

	circleHandlers := circleHTTP.NewHandlers(
		sendConnectionRequestUC,
		acceptConnectionRequestUC,
		declineConnectionRequestUC,
		severConnectionUC,
		blockUserUC,
		unblockUserUC,
		listCircleUC,
		listPendingRequestsUC,
		blockRepo,
		userRepo,
	)

	// --- Host domain wiring ---
	hostRepo := hostPG.NewHostRepository(d.pool)
	createHostProfileUC := hostApp.NewCreateHostProfileUseCase(hostRepo)
	hostHandlers := hostHTTP.NewHandlers(createHostProfileUC)

	// --- Moderator domain wiring ---
	moderatorRepo := moderatorPG.NewModeratorRepository(d.pool)
	createModeratorProfileUC := moderatorApp.NewCreateModeratorProfileUseCase(moderatorRepo)
	moderatorHandlers := moderatorHTTP.NewHandlers(createModeratorProfileUC)

	// --- Activity domain wiring ---
	activityRepo := activityPG.NewActivityRepository(d.pool)

	createActivityUC := activityApp.NewCreateActivityUseCase(activityRepo)
	getActivityUC := activityApp.NewGetActivityUseCase(activityRepo)
	listActivitiesUC := activityApp.NewListActivitiesUseCase(activityRepo)
	listModerationQueueUC := activityApp.NewListModerationQueueUseCase(activityRepo)
	approveActivityUC := activityApp.NewApproveActivityUseCase(activityRepo)
	rejectActivityUC := activityApp.NewRejectActivityUseCase(activityRepo)

	activityHandlers := activityHTTP.NewHandlers(
		createActivityUC,
		getActivityUC,
		listActivitiesUC,
		listModerationQueueUC,
		approveActivityUC,
		rejectActivityUC,
	)

	// --- Hangout domain wiring ---
	hangoutStore := hangoutPG.NewStore(d.pool) // implements domain.Transactor
	hangoutRepo := hangoutPG.NewHangoutRepository(d.pool)
	participantRepo := hangoutPG.NewParticipantRepository(d.pool)
	messageRepo := hangoutPG.NewMessageRepository(d.pool)
	hangoutNotifier := hangoutNotif.NewLogNotifier() // swap for a real adapter once notification infra exists

	createHangoutUC := hangoutApp.NewCreateHangoutUseCase(hangoutRepo, participantRepo, hangoutStore)
	getHangoutUC := hangoutApp.NewGetHangoutUseCase(hangoutRepo, participantRepo)
	listHangoutsUC := hangoutApp.NewListHangoutsUseCase(hangoutRepo)
	inviteParticipantsUC := hangoutApp.NewInviteParticipantsUseCase(hangoutRepo, participantRepo, hangoutNotifier)
	respondToInviteUC := hangoutApp.NewRespondToInviteUseCase(participantRepo, hangoutNotifier)
	cancelHangoutUC := hangoutApp.NewCancelHangoutUseCase(hangoutRepo, participantRepo, hangoutNotifier)
	updateHangoutStatusUC := hangoutApp.NewUpdateHangoutStatusUseCase(hangoutRepo)
	sendMessageUC := hangoutApp.NewSendMessageUseCase(participantRepo, messageRepo)
	listMessagesUC := hangoutApp.NewListMessagesUseCase(participantRepo, messageRepo)

	hangoutHandlers := hangoutHTTP.NewHandlers(
		createHangoutUC,
		getHangoutUC,
		listHangoutsUC,
		inviteParticipantsUC,
		respondToInviteUC,
		cancelHangoutUC,
		updateHangoutStatusUC,
		sendMessageUC,
		listMessagesUC,
	)

	// --- Router ---
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Local development / testing — any loopback address, any port.
			// Covers `python -m http.server` regardless of whether it
			// prints localhost, 127.0.0.1, or 0.0.0.0 in your browser bar.
			if strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "http://0.0.0.0:") {
				return true
			}

			// Real, named origins — your frontend engineer's actual
			// deployed frontend and their local dev server, once known.
			allowed := map[string]bool{
				"https://app.yourdomain.com": true,
				// add their dev server origin here once they give it to you,
				// e.g. "http://localhost:5173" is already covered above,
				// but if they deploy a staging frontend, list its exact origin:
				// "https://staging.yourdomain.com": true,
			}
			return allowed[origin]
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Authorization",
		},
		AllowCredentials: true,
	}))

	router.GET("/tester", func(c *gin.Context) {
		c.Data(
			http.StatusOK,
			"text/html; charset=utf-8",
			apiTesterHTML,
		)
	})

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

	circleGroup := router.Group("/circle")
	circleGroup.Use(authMW)
	circleHandlers.RegisterRoutes(circleGroup)

	activityGroup := router.Group("/activities")
	activityGroup.Use(authMW)
	activityHandlers.RegisterRoutes(activityGroup)

	hangoutGroup := router.Group("/hangouts")
	hangoutGroup.Use(authMW)
	hangoutHandlers.RegisterRoutes(hangoutGroup)

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
