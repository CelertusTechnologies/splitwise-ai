package database

import (
	"context"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/nivra/splitwise-ai/backend/internal/config"
	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	userdomain "github.com/nivra/splitwise-ai/backend/internal/domain/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// devModels are the tables required by the current feature set. Production
// uses the full SQL migration; SQLite dev mode auto-migrates only the models
// that have Go definitions today and adds more as features land.
var devModels = []any{
	&userdomain.User{},
	&authdomain.RefreshToken{},
	&authdomain.OneTimeToken{},
	&authdomain.PhoneOTP{},
	&groupdomain.Group{},
	&groupdomain.Membership{},
	&groupdomain.Invite{},
}

func connectSQLite(ctx context.Context, cfg config.Config, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: newGormLogger(cfg),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// SQLite is a single-writer engine; keep the pool small and give writers a
	// busy timeout so concurrent requests serialize instead of erroring.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := db.WithContext(ctx).Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		return nil, err
	}

	if err := db.WithContext(ctx).AutoMigrate(devModels...); err != nil {
		return nil, err
	}

	log.Info("sqlite connected (development mode)", zap.String("path", cfg.DatabaseURL))
	return db, nil
}
