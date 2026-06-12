package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	sqlDB, err := h.db.DB()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable.")
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable.")
		return
	}

	// Redis is optional in development; when it is wired up, a failed ping is a
	// hard failure so production load balancers drain the instance.
	redisStatus := "disabled"
	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			response.Error(c, http.StatusServiceUnavailable, "redis_unavailable", "Redis is unavailable.")
			return
		}
		redisStatus = "ok"
	}

	response.OK(c, gin.H{
		"status": "ok",
		"redis":  redisStatus,
		"time":   time.Now().UTC(),
	})
}
