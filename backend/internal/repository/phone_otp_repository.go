package repository

import (
	"context"
	"errors"
	"time"

	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PhoneOTPRepository interface {
	Create(ctx context.Context, otp *authdomain.PhoneOTP) error
	FindLatest(ctx context.Context, phoneNumber string) (*authdomain.PhoneOTP, error)
	IncrementAttempt(ctx context.Context, id uuid.UUID) error
	MarkConsumed(ctx context.Context, id uuid.UUID, at time.Time) error
}

type GormPhoneOTPRepository struct {
	db *gorm.DB
}

func NewPhoneOTPRepository(db *gorm.DB) PhoneOTPRepository {
	return &GormPhoneOTPRepository{db: db}
}

func (r *GormPhoneOTPRepository) Create(ctx context.Context, otp *authdomain.PhoneOTP) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *GormPhoneOTPRepository) FindLatest(ctx context.Context, phoneNumber string) (*authdomain.PhoneOTP, error) {
	var found authdomain.PhoneOTP
	err := r.db.WithContext(ctx).
		Where("phone_number = ?", phoneNumber).
		Order("created_at DESC").
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormPhoneOTPRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&authdomain.PhoneOTP{}).
		Where("id = ?", id).
		UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error
}

func (r *GormPhoneOTPRepository) MarkConsumed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authdomain.PhoneOTP{}).
		Where("id = ?", id).
		Update("consumed_at", at).Error
}
