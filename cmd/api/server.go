package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"strings"
	"time"

	recommendationHTTP "github.com/bLorax/khatere-backend/internal/recommendation/adapters/http"
	recommendationPG "github.com/bLorax/khatere-backend/internal/recommendation/adapters/postgres"
	worker "github.com/bLorax/khatere-backend/internal/recommendation/adapters/worker"
	recommendationApp "github.com/bLorax/khatere-backend/internal/recommendation/application"

	badgeHTTP "github.com/bLorax/khatere-backend/internal/badge/adapters/http"
	badgePG "github.com/bLorax/khatere-backend/internal/badge/adapters/postgres"
	badgeApp "github.com/bLorax/khatere-backend/internal/badge/application"

	attendanceHTTP "github.com/bLorax/khatere-backend/internal/attendance/adapters/http"
	attendancePG "github.com/bLorax/khatere-backend/internal/attendance/adapters/postgres"
	attendanceApp "github.com/bLorax/khatere-backend/internal/attendance/application"

	userbadgeHTTP "github.com/bLorax/khatere-backend/internal/userbadge/adapters/http"
	userbadgePG "github.com/bLorax/khatere-backend/internal/userbadge/adapters/postgres"
	userbadgeApp "github.com/bLorax/khatere-backend/internal/userbadge/application"

	deepseek "github.com/bLorax/khatere-backend/internal/comment/adapters/deepseek"
	gapgpt "github.com/bLorax/khatere-backend/internal/comment/adapters/gapgpt"
	commentHTTP "github.com/bLorax/khatere-backend/internal/comment/adapters/http"
	commentPG "github.com/bLorax/khatere-backend/internal/comment/adapters/postgres"
	commentApp "github.com/bLorax/khatere-backend/internal/comment/application"
	commentDomain "github.com/bLorax/khatere-backend/internal/comment/domain"

	archivegateway "github.com/bLorax/khatere-backend/internal/archive/adapters/hangout"
	archiveHTTP "github.com/bLorax/khatere-backend/internal/archive/adapters/http"
	archiveminio "github.com/bLorax/khatere-backend/internal/archive/adapters/minio"
	archivePG "github.com/bLorax/khatere-backend/internal/archive/adapters/postgres"
	archiveApp "github.com/bLorax/khatere-backend/internal/archive/application"
	archivecreator "github.com/bLorax/khatere-backend/internal/hangout/adapters/archive"
	suggestionacceptance "github.com/bLorax/khatere-backend/internal/hangout/adapters/recommendation"

	activityHTTP "github.com/bLorax/khatere-backend/internal/activity/adapters/http"
	activityPG "github.com/bLorax/khatere-backend/internal/activity/adapters/postgres"
	activityApp "github.com/bLorax/khatere-backend/internal/activity/application"

	ratingHTTP "github.com/bLorax/khatere-backend/internal/rating/adapters/http"
	ratingPG "github.com/bLorax/khatere-backend/internal/rating/adapters/postgres"
	ratingApp "github.com/bLorax/khatere-backend/internal/rating/application"

	hangoutWS "github.com/bLorax/khatere-backend/internal/hangout/adapters/websocket"
	qrcodeHTTP "github.com/bLorax/khatere-backend/internal/qrcode/adapters/http"
	qrcodePG "github.com/bLorax/khatere-backend/internal/qrcode/adapters/postgres"
	qrcodeApp "github.com/bLorax/khatere-backend/internal/qrcode/application"

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
	cfg               *config.Config
	pool              *pgxpool.Pool
	redisClient       *goredis.Client
	minioClient       *miniogo.Client
	minioPublicClient *miniogo.Client
}

// newRouter wires every bounded context's repositories, use cases, and
// HTTP handlers, then returns a ready-to-run gin.Engine, plus the
// recommendation cache-refresh worker (Step 5) so main.go can start
// and stop it alongside the server.
func newRouter(d deps) (*gin.Engine, *worker.RefreshWorker) {
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
	getProfileUC := userApp.NewGetProfileUseCase(userRepo)
	updateProfileUC := userApp.NewUpdateProfileUseCase(userRepo)
	setInterestsUC := userApp.NewSetInterestsUseCase(interestRepo)
	listInterestsUC := userApp.NewListInterestsUseCase(interestRepo)
	listCatalogUC := userApp.NewListInterestCatalogUseCase(interestRepo)

	userHandlers := userHTTP.NewHandlers(createProfileUC, getProfileUC, updateProfileUC, setInterestsUC, listInterestsUC, listCatalogUC)
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
	getHostProfileUC := hostApp.NewGetHostProfileUseCase(hostRepo)
	hostHandlers := hostHTTP.NewHandlers(createHostProfileUC, getHostProfileUC)

	// --- Moderator domain wiring ---
	moderatorRepo := moderatorPG.NewModeratorRepository(d.pool)
	createModeratorProfileUC := moderatorApp.NewCreateModeratorProfileUseCase(moderatorRepo)
	getModeratorProfileUC := moderatorApp.NewGetModeratorProfileUseCase(moderatorRepo)
	moderatorHandlers := moderatorHTTP.NewHandlers(createModeratorProfileUC, getModeratorProfileUC)

	// --- Activity domain wiring ---
	activityRepo := activityPG.NewActivityRepository(d.pool)
	activityInterestRepo := activityPG.NewActivityInterestRepository(d.pool)

	createActivityUC := activityApp.NewCreateActivityUseCase(activityRepo)
	getActivityUC := activityApp.NewGetActivityUseCase(activityRepo)
	listActivitiesUC := activityApp.NewListActivitiesUseCase(activityRepo)
	listMyActivitiesUC := activityApp.NewListMyActivitiesUseCase(activityRepo)
	listModerationQueueUC := activityApp.NewListModerationQueueUseCase(activityRepo)
	approveActivityUC := activityApp.NewApproveActivityUseCase(activityRepo)
	rejectActivityUC := activityApp.NewRejectActivityUseCase(activityRepo)
	setActivityInterestsUC := activityApp.NewSetActivityInterestsUseCase(activityRepo, activityInterestRepo)
	listActivityInterestsUC := activityApp.NewListActivityInterestsUseCase(activityInterestRepo)

	activityHandlers := activityHTTP.NewHandlers(
		createActivityUC,
		getActivityUC,
		listActivitiesUC,
		listMyActivitiesUC,
		listModerationQueueUC,
		approveActivityUC,
		rejectActivityUC,
		setActivityInterestsUC,
		listActivityInterestsUC,
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

	// --- QRCode domain wiring ---
	// activityRepo satisfies qrcodeApp.ActivityOwnershipChecker via
	// its IsOwnedByHost method — see internal/activity/adapters/postgres.
	qrCodeRepo := qrcodePG.NewQRCodeRepository(d.pool)
	generateQRCodeUC := qrcodeApp.NewGenerateQRCodeUseCase(qrCodeRepo, activityRepo)
	getQRCodeUC := qrcodeApp.NewGetQRCodeUseCase(qrCodeRepo, activityRepo)
	qrCodeHandlers := qrcodeHTTP.NewHandlers(generateQRCodeUC, getQRCodeUC)

	// --- Badge domain wiring ---
	// activityRepo satisfies badgeApp.ActivityOwnershipChecker via
	// its IsOwnedByHost method, same as the qrcode domain.
	badgeRepo := badgePG.NewBadgeRepository(d.pool)
	createBadgeUC := badgeApp.NewCreateBadgeUseCase(badgeRepo, activityRepo)
	getBadgeUC := badgeApp.NewGetBadgeUseCase(badgeRepo, activityRepo)
	badgeHandlers := badgeHTTP.NewHandlers(createBadgeUC, getBadgeUC)

	// --- UserBadge domain wiring ---
	// badgeLookupRepo is a purpose-built adapter satisfying
	// userbadgeApp.BadgeLookup, so this domain never imports
	// badge/domain directly — see badge_lookup_repository.go.
	userBadgeRepo := userbadgePG.NewUserBadgeRepository(d.pool)
	badgeLookupRepo := userbadgePG.NewBadgeLookupRepository(d.pool)
	awardBadgeUC := userbadgeApp.NewAwardBadgeUseCase(userBadgeRepo, badgeLookupRepo)
	listUserBadgesUC := userbadgeApp.NewListUserBadgesUseCase(userBadgeRepo)
	userBadgeHandlers := userbadgeHTTP.NewHandlers(listUserBadgesUC)

	// --- Attendance domain wiring ---
	// awardBadgeUC satisfies attendanceApp.BadgeAwarder via its
	// AwardIfBadgeExists method. qrCodeLookupRepo is the
	// attendance package's own ResolveCode adapter over the
	// qr_codes table — not the qrcode domain's repository.
	attendanceRepo := attendancePG.NewAttendanceVerificationRepository(d.pool)
	qrCodeLookupRepo := attendancePG.NewQRCodeRepository(d.pool)
	verifyScanUC := attendanceApp.NewVerifyScanUseCase(attendanceRepo, qrCodeLookupRepo, awardBadgeUC)
	attendanceHandlers := attendanceHTTP.NewHandlers(verifyScanUC)

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
	listCommentsUC := commentApp.NewListCommentsUseCase(
		commentRepo,
		commentVoteRepo,
		attendanceRepo,
	)
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

	// --- Recommendation domain wiring ---
	recommendationCache := recommendationPG.NewRecommendationRepository(d.pool)
	candidateSource := recommendationPG.NewActivityCandidateSource(d.pool)
	interestSource := recommendationPG.NewInterestSource(d.pool)
	historySource := recommendationPG.NewHistorySource(d.pool)
	engagementSource := recommendationPG.NewEngagementSource(d.pool)
	qualitySource := recommendationPG.NewQualitySource(d.pool)
	suggestionStatsRepo := recommendationPG.NewSuggestionStatsRepository(d.pool)

	generateSuggestionsUC := recommendationApp.NewGenerateSuggestionsUseCase(
		candidateSource, interestSource, historySource, engagementSource, qualitySource, suggestionStatsRepo, recommendationCache,
	)
	getSuggestionsUC := recommendationApp.NewGetSuggestionsUseCase(recommendationCache, generateSuggestionsUC, suggestionStatsRepo)
	recommendationHandlers := recommendationHTTP.NewHandlers(getSuggestionsUC, generateSuggestionsUC)

	// Background cache-refresh worker (Step 5, side A — periodic,
	// not event-triggered). main.go starts this in its own
	// goroutine and stops it on shutdown.
	recommendationWorker := worker.New(
		recommendationCache,
		generateSuggestionsUC,
		d.cfg.RecommendationRefreshInterval,
		d.cfg.RecommendationStaleAfter,
	)

	// --- Archive domain wiring ---
	archiveStore := archivePG.NewStore(d.pool) // implements domain.Transactor
	archiveRepo := archivePG.NewArchiveRepository(d.pool)
	archiveMediaRepo := archivePG.NewArchiveMediaRepository(d.pool)
	deletionMarkRepo := archivePG.NewDeletionMarkRepository(d.pool)

	archiveHangoutGateway := archivegateway.New(hangoutRepo, participantRepo, messageRepo)
	archiveStorage := archiveminio.New(d.minioClient, d.minioPublicClient, d.cfg.ArchiveMediaBucket)
	createArchiveUC := archiveApp.NewCreateArchiveUseCase(archiveRepo, archiveHangoutGateway)

	// Mirror-image gateway: lets Hangout trigger archive creation on
	// cancel/complete without importing the archive module directly.
	hangoutArchiveCreator := archivecreator.New(createArchiveUC)

	// Same mirror-image shape as hangoutArchiveCreator above: lets
	// Hangout tell Recommendation about an accepted suggestion
	// without importing that module directly.
	hangoutSuggestionAcceptance := suggestionacceptance.New(suggestionStatsRepo)

	// --- Hangout domain wiring (use cases + handlers) ---
	createHangoutUC := hangoutApp.NewCreateHangoutUseCase(hangoutRepo, participantRepo, hangoutStore, hangoutSuggestionAcceptance)
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

	chatHub := hangoutWS.NewHub()

	hangoutHandlers := hangoutHTTP.NewHandlers(
		createHangoutUC, getHangoutUC, listHangoutsUC,
		inviteParticipantsUC, respondToInviteUC, cancelHangoutUC,
		updateHangoutStatusUC, sendMessageUC, listMessagesUC,
		proposeMeetupPinUC, respondToMeetupPinUC, getMeetupPinUC,
		chatHub,
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
		deleteArchiveUC, listPendingPromptsUC, archiveStorage,
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
	userBadgeHandlers.RegisterRoutes(userGroup)

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

	qrCodeHandlers.RegisterRoutes(ratingGroup) // reuse activityGroup.Group("/:id")
	badgeHandlers.RegisterRoutes(ratingGroup)

	commentHandlers.RegisterRoutes(ratingGroup) // reuse activityGroup.Group("/:id")

	commentVoteGroup := router.Group("/comment")
	commentVoteGroup.Use(authMW)
	commentHandlers.RegisterVoteRoute(commentVoteGroup)

	commentHandlers.RegisterModerationRoutes(moderatorGroup)

	hangoutGroup := router.Group("/hangouts")
	hangoutGroup.Use(authMW)
	hangoutHandlers.RegisterRoutes(hangoutGroup)
	hangoutChatGroup := router.Group("/hangouts")
	hangoutChatGroup.Use(middleware.AuthRequiredQuery(d.cfg.JWTSecret))
	hangoutHandlers.RegisterChatSocketRoute(hangoutChatGroup)

	attendanceGroup := router.Group("/attendance")
	attendanceGroup.Use(authMW)
	attendanceHandlers.RegisterRoutes(attendanceGroup)

	recommendationGroup := router.Group("/recommendations")
	recommendationGroup.Use(authMW)
	recommendationHandlers.RegisterRoutes(recommendationGroup)

	archiveGroup := router.Group("/archives")
	archiveGroup.Use(authMW)
	archiveHandlers.RegisterRoutes(archiveGroup)

	// Lives on the hangout group intentionally — see RegisterUploadPromptRoute's
	// doc comment in internal/archive/adapters/http/handler.go.
	archiveHandlers.RegisterUploadPromptRoute(hangoutGroup)

	return router, recommendationWorker
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
