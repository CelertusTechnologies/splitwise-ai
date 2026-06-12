package database

import (
	"context"
	"fmt"

	"github.com/nivra/splitwise-ai/backend/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Connect opens a database connection for the driver configured in cfg.
//
// Production runs on PostgreSQL (schema managed by SQL migrations). Local
// development can run on an embedded, pure-Go SQLite file so the API boots with
// zero external services; in that mode the schema is created via GORM
// auto-migration of the domain models.
func Connect(ctx context.Context, cfg config.Config, log *zap.Logger) (*gorm.DB, error) {
	switch cfg.DBDriver {
	case config.DriverSQLite:
		return connectSQLite(ctx, cfg, log)
	case config.DriverPostgres, "":
		return connectPostgres(ctx, cfg, log)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (use %q or %q)", cfg.DBDriver, config.DriverPostgres, config.DriverSQLite)
	}
}

func newGormLogger(cfg config.Config) gormlogger.Interface {
	level := gormlogger.Warn
	if cfg.IsProduction() {
		level = gormlogger.Silent
	}
	return gormlogger.Default.LogMode(level)
}
