package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	userdomain "github.com/nivra/splitwise-ai/backend/internal/domain/user"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *userdomain.User) error
	FindByEmail(ctx context.Context, email string) (*userdomain.User, error)
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*userdomain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID, at time.Time) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *userdomain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*userdomain.User, error) {
	var found userdomain.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormUserRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*userdomain.User, error) {
	var found userdomain.User
	err := r.db.WithContext(ctx).
		Where("phone_number = ? AND deleted_at IS NULL", phoneNumber).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	var found userdomain.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&userdomain.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_login_at": at,
			"updated_at":    at,
		}).Error
}

func (r *GormUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&userdomain.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"updated_at":    now,
		}).Error
}

func (r *GormUserRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&userdomain.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"email_verified_at": at,
			"status":            userdomain.StatusActive,
			"updated_at":        at,
		}).Error
}
