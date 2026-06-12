package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *authdomain.RefreshToken) error
	FindActiveByTokenID(ctx context.Context, tokenID string, now time.Time) (*authdomain.RefreshToken, error)
	RevokeByTokenID(ctx context.Context, tokenID string, replacementTokenID *string, revokedAt time.Time) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type GormRefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &GormRefreshTokenRepository{db: db}
}

func (r *GormRefreshTokenRepository) Create(ctx context.Context, token *authdomain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormRefreshTokenRepository) FindActiveByTokenID(ctx context.Context, tokenID string, now time.Time) (*authdomain.RefreshToken, error) {
	var found authdomain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_id = ? AND revoked_at IS NULL AND expires_at > ?", tokenID, now).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormRefreshTokenRepository) RevokeByTokenID(ctx context.Context, tokenID string, replacementTokenID *string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authdomain.RefreshToken{}).
		Where("token_id = ? AND revoked_at IS NULL", tokenID).
		Updates(map[string]any{
			"revoked_at":           revokedAt,
			"replaced_by_token_id": replacementTokenID,
		}).Error
}

func (r *GormRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authdomain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", revokedAt).Error
}
