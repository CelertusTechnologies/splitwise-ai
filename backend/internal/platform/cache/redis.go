package cache

import (
	"context"
	"time"

	"github.com/nivra/splitwise-ai/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func Connect(ctx context.Context, cfg config.Config, log *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, err
	}

	log.Info("redis connected")
	return client, nil
}
