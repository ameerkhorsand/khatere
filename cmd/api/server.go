package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"strings"
	"time"

	deepseek "github.com/bLorax/khatere-backend/internal/comment/adapters/deepseek"
	gapgpt "github.com/bLorax/khatere-backend/internal/comment/adapters/gapgpt"
	commentHTTP "github.com/bLorax/khatere-backend/internal/comment/adapters/http"
	commentPG "github.com/bLorax/khatere-backend/internal/comment/adapters/postgres"
	commentApp "github.com/bLorax/khatere-backend/internal/comment/application"
	commentDomain "github.com/bLorax/khatere-backend/internal/comment/domain"

	archivecreator "github.com/bLorax/khatere-backend/internal/hangout/adapters/archive"

	archivegateway "github.com/bLorax/khatere-backend/internal/archive/adapters/hangout"
	archiveHTTP "github.com/bLorax/khatere-backend/internal/archive/adapters/http"
	archiveminio "github.com/bLorax/khatere-backend/internal/archive/adapters/minio"
	archivePG "github.com/bLorax/khatere-backend/internal/archive/adapters/postgres"
	archiveApp "github.com/bLorax/khatere-backend/internal/archive/application"

	activityHTTP "github.com/bLorax/khatere-backend/internal/activity/adapters/http"
	activityPG "github.com/bLorax/khatere-backend/internal/activity/adapters/postgres"
	activityApp "github.com/bLorax/khatere-backend/internal/activity/application"

	ratingHTTP "github.com/bLorax/khatere-backend/internal/rating/adapters/http"
	ratingPG "github.com/bLorax/khatere-backend/internal/rating/adapters/postgres"
	ratingApp "github.com/bLorax/khatere-backend/internal/rating/application"

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
	userPublicHandlers := userHTTP.NewPublicHandlers(userRepo)

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

	// --- Hangout domain wiring (repositories + infra only, for now) ---
	hangoutStore := hangoutPG.NewStore(d.pool) // implements domain.Transactor
	hangoutRepo := hangoutPG.NewHangoutRepository(d.pool)
	participantRepo := hangoutPG.NewParticipantRepository(d.pool)
	messageRepo := hangoutPG.NewMessageRepository(d.pool)
	meetupPinRepo := hangoutPG.NewMeetupPinRepository(d.pool)
	pinConfirmationRepo := hangoutPG.NewPinConfirmationRepository(d.pool)
	hangoutNotifier := hangoutNotif.NewLogNotifier()

	// --- Rating domain wiring ---
	// hangoutRepo satisfies ratingApp.AttendanceChecker via its
	// Attended method — see internal/rating/application/create_rating.go.
	ratingRepo := ratingPG.NewRatingRepository(d.pool)
	createRatingUC := ratingApp.NewCreateRatingUseCase(ratingRepo, hangoutRepo)
	getRatingSummaryUC := ratingApp.NewGetRatingSummaryUseCase(ratingRepo)
	ratingHandlers := ratingHTTP.NewHandlers(createRatingUC, getRatingSummaryUC)

	// --- Comment domain wiring ---
	commentRepo := commentPG.NewCommentRepository(d.pool)
	commentVoteRepo := commentPG.NewCommentVoteRepository(d.pool)
	commentSummaryRepo := commentPG.NewCommentSummaryRepository(d.pool)

	// AI comment summaries are optional: with no key configured for
	// either vendor, Disabled is wired in instead of a real client.
	// This degrades one feature (the summary endpoint returns
	// available:false, approving a comment logs a warning) rather
	// than stopping the whole server from starting over a
	// third-party key nobody set yet. gapGPT takes priority if both
	// happen to be set, since it's the one currently in active use.
	var commentSummarizer commentDomain.CommentSummarizer
	switch {
	case d.cfg.GapGPTAPIKey != "":
		commentSummarizer = gapgpt.NewClient(d.cfg.GapGPTAPIKey, d.cfg.GapGPTModel)
	case d.cfg.DeepSeekAPIKey != "":
		commentSummarizer = deepseek.NewClient(d.cfg.DeepSeekAPIKey)
	default:
		commentSummarizer = deepseek.Disabled{}
		log.Println("comment: no AI summarizer key set (GAPGPT_API_KEY or DEEPSEEK_API_KEY) — AI comment summaries are disabled")
	}

	createCommentUC := commentApp.NewCreateCommentUseCase(commentRepo)
	listCommentsUC := commentApp.NewListCommentsUseCase(commentRepo, commentVoteRepo)
	voteCommentUC := commentApp.NewVoteCommentUseCase(commentRepo, commentVoteRepo)
	listCommentModerationUC := commentApp.NewListCommentModerationQueueUseCase(commentRepo)
	regenerateSummaryUC := commentApp.NewRegenerateCommentSummaryUseCase(commentRepo, commentSummaryRepo, commentSummarizer)
	getSummaryUC := commentApp.NewGetCommentSummaryUseCase(commentSummaryRepo)
	approveCommentUC := commentApp.NewApproveCommentUseCase(commentRepo, regenerateSummaryUC)
	rejectCommentUC := commentApp.NewRejectCommentUseCase(commentRepo)

	commentHandlers := commentHTTP.NewHandlers(
		createCommentUC, listCommentsUC, voteCommentUC,
		listCommentModerationUC, approveCommentUC, rejectCommentUC,
		getSummaryUC,
	)

	// --- Archive domain wiring ---
	archiveStore := archivePG.NewStore(d.pool) // implements domain.Transactor
	archiveRepo := archivePG.NewArchiveRepository(d.pool)
	archiveMediaRepo := archivePG.NewArchiveMediaRepository(d.pool)
	deletionMarkRepo := archivePG.NewDeletionMarkRepository(d.pool)

	archiveHangoutGateway := archivegateway.New(hangoutRepo, participantRepo, messageRepo)
	archiveStorage := archiveminio.New(d.minioClient, d.cfg.ArchiveMediaBucket)
	createArchiveUC := archiveApp.NewCreateArchiveUseCase(archiveRepo, archiveHangoutGateway)

	// Mirror-image gateway: lets Hangout trigger archive creation on
	// cancel/complete without importing the archive module directly.
	hangoutArchiveCreator := archivecreator.New(createArchiveUC)

	// --- Hangout domain wiring (use cases + handlers) ---
	createHangoutUC := hangoutApp.NewCreateHangoutUseCase(hangoutRepo, participantRepo, hangoutStore)
	getHangoutUC := hangoutApp.NewGetHangoutUseCase(hangoutRepo, participantRepo)
	listHangoutsUC := hangoutApp.NewListHangoutsUseCase(hangoutRepo)
	inviteParticipantsUC := hangoutApp.NewInviteParticipantsUseCase(hangoutRepo, participantRepo, hangoutNotifier)
	respondToInviteUC := hangoutApp.NewRespondToInviteUseCase(participantRepo, hangoutNotifier)
	cancelHangoutUC := hangoutApp.NewCancelHangoutUseCase(hangoutRepo, participantRepo, hangoutNotifier, hangoutArchiveCreator)
	updateHangoutStatusUC := hangoutApp.NewUpdateHangoutStatusUseCase(hangoutRepo, hangoutArchiveCreator)
	sendMessageUC := hangoutApp.NewSendMessageUseCase(participantRepo, messageRepo)
	listMessagesUC := hangoutApp.NewListMessagesUseCase(participantRepo, messageRepo)
	proposeMeetupPinUC := hangoutApp.NewProposeMeetupPinUseCase(hangoutRepo, participantRepo, meetupPinRepo, pinConfirmationRepo, hangoutNotifier, hangoutStore)
	respondToMeetupPinUC := hangoutApp.NewRespondToMeetupPinUseCase(hangoutRepo, meetupPinRepo, pinConfirmationRepo, hangoutNotifier)
	getMeetupPinUC := hangoutApp.NewGetMeetupPinUseCase(participantRepo, meetupPinRepo, pinConfirmationRepo)

	hangoutHandlers := hangoutHTTP.NewHandlers(
		createHangoutUC, getHangoutUC, listHangoutsUC,
		inviteParticipantsUC, respondToInviteUC, cancelHangoutUC,
		updateHangoutStatusUC, sendMessageUC, listMessagesUC,
		proposeMeetupPinUC, respondToMeetupPinUC, getMeetupPinUC,
	)

	// --- Archive domain wiring (remaining use cases + handlers) ---
	listArchivesUC := archiveApp.NewListArchivesUseCase(archiveRepo, archiveHangoutGateway)
	getArchiveUC := archiveApp.NewGetArchiveUseCase(archiveRepo, archiveMediaRepo, archiveHangoutGateway)
	uploadMediaUC := archiveApp.NewUploadMediaUseCase(archiveRepo, archiveMediaRepo, archiveHangoutGateway, archiveStorage)
	markArchiveDeletedUC := archiveApp.NewMarkArchiveDeletedUseCase(archiveRepo, archiveHangoutGateway, deletionMarkRepo)
	purgeArchiveUC := archiveApp.NewCheckAndPurgeArchiveUseCase(archiveRepo, archiveMediaRepo, archiveHangoutGateway, deletionMarkRepo, archiveStorage)
	deleteArchiveUC := archiveApp.NewDeleteArchiveUseCase(archiveStore, markArchiveDeletedUC, purgeArchiveUC)
	listPendingPromptsUC := archiveApp.NewListPendingUploadPromptsUseCase(archiveHangoutGateway, archiveRepo)

	archiveHandlers := archiveHTTP.NewHandlers(
		listArchivesUC, getArchiveUC, uploadMediaUC,
		deleteArchiveUC, listPendingPromptsUC,
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

	// Public, read-only lookups (e.g. resolving a UUID to a name for
	// Circle/Hangout screens). Separate from /user on purpose — this
	// group returns other people's data, not the caller's own.
	usersGroup := router.Group("/users")
	usersGroup.Use(authMW)
	userPublicHandlers.RegisterRoutes(usersGroup)

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

	// Rating routes nest under /activities/:id — RegisterRoutes reads
	// :id from this parent group, same as GetActivity does.
	ratingGroup := activityGroup.Group("/:id")
	ratingHandlers.RegisterRoutes(ratingGroup)

	commentHandlers.RegisterRoutes(ratingGroup) // reuse activityGroup.Group("/:id")

	commentVoteGroup := router.Group("/comment")
	commentVoteGroup.Use(authMW)
	commentHandlers.RegisterVoteRoute(commentVoteGroup)

	commentHandlers.RegisterModerationRoutes(moderatorGroup)

	hangoutGroup := router.Group("/hangouts")
	hangoutGroup.Use(authMW)
	hangoutHandlers.RegisterRoutes(hangoutGroup)

	archiveGroup := router.Group("/archives")
	archiveGroup.Use(authMW)
	archiveHandlers.RegisterRoutes(archiveGroup)

	// Lives on the hangout group intentionally — see RegisterUploadPromptRoute's
	// doc comment in internal/archive/adapters/http/handler.go.
	archiveHandlers.RegisterUploadPromptRoute(hangoutGroup)

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
