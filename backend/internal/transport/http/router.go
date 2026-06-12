package httptransport

import (
	"github.com/gin-gonic/gin"
	"github.com/nivra/splitwise-ai/backend/internal/config"
	"github.com/nivra/splitwise-ai/backend/internal/platform/security"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/handlers"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/middleware"
	"go.uber.org/zap"
)

type RouterDeps struct {
	Config        config.Config
	Logger        *zap.Logger
	JWTManager    *security.JWTManager
	AuthHandler   *handlers.AuthHandler
	MeHandler     *handlers.MeHandler
	HealthHandler *handlers.HealthHandler
}

func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.Config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(deps.Logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(deps.Config.FrontendURL))

	router.GET("/health", deps.HealthHandler.Check)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/signup", deps.AuthHandler.SignUp)
			auth.POST("/login", deps.AuthHandler.Login)
			auth.POST("/refresh", deps.AuthHandler.Refresh)
			auth.POST("/logout", deps.AuthHandler.Logout)
			auth.POST("/forgot-password", deps.AuthHandler.ForgotPassword)
			auth.POST("/reset-password", deps.AuthHandler.ResetPassword)
			auth.POST("/verify-email", deps.AuthHandler.VerifyEmail)
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthRequired(deps.JWTManager))
		{
			users.GET("/me", deps.MeHandler.GetMe)
		}
	}

	return router
}
