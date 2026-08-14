package repository

import (
	"context"
	"errors"

	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupInviteRepository interface {
	Create(ctx context.Context, invite *groupdomain.Invite) error
	FindByCode(ctx context.Context, code string) (*groupdomain.Invite, error)
	IncrementUse(ctx context.Context, id uuid.UUID) error
}

type GormGroupInviteRepository struct {
	db *gorm.DB
}

func NewGroupInviteRepository(db *gorm.DB) GroupInviteRepository {
	return &GormGroupInviteRepository{db: db}
}

func (r *GormGroupInviteRepository) Create(ctx context.Context, invite *groupdomain.Invite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *GormGroupInviteRepository) FindByCode(ctx context.Context, code string) (*groupdomain.Invite, error) {
	var found groupdomain.Invite
	err := r.db.WithContext(ctx).
		Where("invite_code = ?", code).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormGroupInviteRepository) IncrementUse(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&groupdomain.Invite{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}
