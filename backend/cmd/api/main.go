package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/nivra/splitwise-ai/backend/internal/config"
	"github.com/nivra/splitwise-ai/backend/internal/platform/cache"
	"github.com/nivra/splitwise-ai/backend/internal/platform/database"
	applogger "github.com/nivra/splitwise-ai/backend/internal/platform/logger"
	"github.com/nivra/splitwise-ai/backend/internal/platform/security"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	httptransport "github.com/nivra/splitwise-ai/backend/internal/transport/http"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/handlers"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	// Best-effort load of a local .env so `go run ./cmd/api` picks up dev config.
	// Existing process env always wins and godotenv never overrides already-set
	// vars, so loading each candidate independently is safe. Covers running from
	// the repo root or from backend/; missing files are ignored (e.g. Docker,
	// which injects env directly).
	for _, envFile := range []string{".env", "../.env"} {
		_ = godotenv.Load(envFile)
	}

	cfg := config.Load()

	logger, err := applogger.New(cfg.Env)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	db, err := database.Connect(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err), zap.String("driver", cfg.DBDriver))
	}

	// Redis is optional in development (currently only backing health checks and
	// future rate-limiting). In production a failed connection is fatal.
	redisClient, err := cache.Connect(ctx, cfg, logger)
	if err != nil {
		if cfg.IsProduction() {
			logger.Fatal("failed to connect redis", zap.Error(err))
		}
		logger.Warn("redis unavailable, continuing without cache (development mode)", zap.Error(err))
		redisClient = nil
	}

	jwtManager := security.NewJWTManager(cfg)
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	oneTimeTokenRepo := repository.NewOneTimeTokenRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	groupMembershipRepo := repository.NewGroupMembershipRepository(db)
	groupInviteRepo := repository.NewGroupInviteRepository(db)
	phoneOTPRepo := repository.NewPhoneOTPRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	expenseShareRepo := repository.NewExpenseShareRepository(db)
	expenseCategoryRepo := repository.NewExpenseCategoryRepository(db)

	authService := service.NewAuthService(cfg, userRepo, refreshTokenRepo, oneTimeTokenRepo, jwtManager)
	groupService := service.NewGroupService(groupRepo, groupMembershipRepo, groupInviteRepo, userRepo)
	phoneOTPService := service.NewPhoneOTPService(cfg, userRepo, phoneOTPRepo, authService)
	expenseService := service.NewExpenseService(groupRepo, groupMembershipRepo, expenseCategoryRepo, expenseRepo, expenseShareRepo)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		Config:          cfg,
		Logger:          logger,
		JWTManager:      jwtManager,
		AuthHandler:     handlers.NewAuthHandler(authService),
		MeHandler:       handlers.NewMeHandler(userRepo),
		HealthHandler:   handlers.NewHealthHandler(db, redisClient),
		GroupHandler:    handlers.NewGroupHandler(groupService, cfg.FrontendURL),
		PhoneOTPHandler: handlers.NewPhoneOTPHandler(phoneOTPService),
		ExpenseHandler:  handlers.NewExpenseHandler(expenseService),
	})

	logger.Info("starting api", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Fatal("api stopped", zap.Error(err))
	}
}
