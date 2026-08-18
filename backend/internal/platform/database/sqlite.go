package database

import (
	"context"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/nivra/splitwise-ai/backend/internal/config"
	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
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
	&expensedomain.Expense{},
	&expensedomain.Share{},
	&expensedomain.Category{},
}

// devExpenseCategories mirrors the seed data inserted by the Postgres SQL
// migration, since SQLite dev mode only auto-migrates schema, not seed rows.
var devExpenseCategories = []expensedomain.Category{
	{Slug: "food", Name: "Food", Icon: "utensils"},
	{Slug: "travel", Name: "Travel", Icon: "plane"},
	{Slug: "hotel", Name: "Hotel", Icon: "bed"},
	{Slug: "shopping", Name: "Shopping", Icon: "shopping-bag"},
	{Slug: "entertainment", Name: "Entertainment", Icon: "ticket"},
	{Slug: "utilities", Name: "Utilities", Icon: "bolt"},
	{Slug: "fuel", Name: "Fuel", Icon: "fuel"},
	{Slug: "miscellaneous", Name: "Miscellaneous", Icon: "circle-ellipsis"},
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

	var categoryCount int64
	if err := db.WithContext(ctx).Model(&expensedomain.Category{}).Count(&categoryCount).Error; err != nil {
		return nil, err
	}
	if categoryCount == 0 {
		if err := db.WithContext(ctx).Create(&devExpenseCategories).Error; err != nil {
			return nil, err
		}
	}

	log.Info("sqlite connected (development mode)", zap.String("path", cfg.DatabaseURL))
	return db, nil
}
