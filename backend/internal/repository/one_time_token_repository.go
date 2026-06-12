package repository

import (
	"context"
	"errors"
	"time"

	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	"gorm.io/gorm"
)

type OneTimeTokenRepository interface {
	Create(ctx context.Context, token *authdomain.OneTimeToken) error
	FindActiveByHash(ctx context.Context, purpose, tokenHash string, now time.Time) (*authdomain.OneTimeToken, error)
	MarkUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
}

type GormOneTimeTokenRepository struct {
	db *gorm.DB
}

func NewOneTimeTokenRepository(db *gorm.DB) OneTimeTokenRepository {
	return &GormOneTimeTokenRepository{db: db}
}

func (r *GormOneTimeTokenRepository) Create(ctx context.Context, token *authdomain.OneTimeToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormOneTimeTokenRepository) FindActiveByHash(ctx context.Context, purpose, tokenHash string, now time.Time) (*authdomain.OneTimeToken, error) {
	var found authdomain.OneTimeToken
	err := r.db.WithContext(ctx).
		Where("purpose = ? AND token_hash = ? AND used_at IS NULL AND expires_at > ?", purpose, tokenHash, now).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormOneTimeTokenRepository) MarkUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authdomain.OneTimeToken{}).
		Where("token_hash = ? AND used_at IS NULL", tokenHash).
		Update("used_at", usedAt).Error
}
