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
	Config          config.Config
	Logger          *zap.Logger
	JWTManager      *security.JWTManager
	AuthHandler     *handlers.AuthHandler
	MeHandler       *handlers.MeHandler
	HealthHandler   *handlers.HealthHandler
	GroupHandler    *handlers.GroupHandler
	PhoneOTPHandler *handlers.PhoneOTPHandler
	ExpenseHandler  *handlers.ExpenseHandler
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

			otp := auth.Group("/otp")
			{
				otp.POST("/request", deps.PhoneOTPHandler.Request)
				otp.POST("/verify", deps.PhoneOTPHandler.Verify)
				otp.POST("/complete-signup", deps.PhoneOTPHandler.CompleteSignup)
			}
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthRequired(deps.JWTManager))
		{
			users.GET("/me", deps.MeHandler.GetMe)
		}

		groups := v1.Group("/groups")
		groups.Use(middleware.AuthRequired(deps.JWTManager))
		{
			groups.POST("", deps.GroupHandler.Create)
			groups.GET("", deps.GroupHandler.List)
			groups.POST("/join", deps.GroupHandler.Join)
			groups.GET("/:id", deps.GroupHandler.Get)
			groups.GET("/:id/members", deps.GroupHandler.ListMembers)
			groups.POST("/:id/invites", deps.GroupHandler.CreateInvite)
			groups.POST("/:id/expenses", deps.ExpenseHandler.Create)
			groups.GET("/:id/expenses", deps.ExpenseHandler.List)
			groups.GET("/:id/balances", deps.ExpenseHandler.Balances)
		}

		expenses := v1.Group("/expenses")
		expenses.Use(middleware.AuthRequired(deps.JWTManager))
		{
			expenses.DELETE("/:id", deps.ExpenseHandler.Delete)
		}

		expenseCategories := v1.Group("/expense-categories")
		expenseCategories.Use(middleware.AuthRequired(deps.JWTManager))
		{
			expenseCategories.GET("", deps.ExpenseHandler.ListCategories)
		}
	}

	return router
}
