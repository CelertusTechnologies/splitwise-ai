package repository

import (
	"context"
	"errors"

	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupMembershipRepository interface {
	Create(ctx context.Context, m *groupdomain.Membership) error
	FindActive(ctx context.Context, groupID, userID uuid.UUID) (*groupdomain.Membership, error)
	ListActiveGroupIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	CountActive(ctx context.Context, groupID uuid.UUID) (int64, error)
}

type GormGroupMembershipRepository struct {
	db *gorm.DB
}

func NewGroupMembershipRepository(db *gorm.DB) GroupMembershipRepository {
	return &GormGroupMembershipRepository{db: db}
}

func (r *GormGroupMembershipRepository) Create(ctx context.Context, m *groupdomain.Membership) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *GormGroupMembershipRepository) FindActive(ctx context.Context, groupID, userID uuid.UUID) (*groupdomain.Membership, error) {
	var found groupdomain.Membership
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, groupdomain.MembershipStatusActive).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormGroupMembershipRepository) ListActiveGroupIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&groupdomain.Membership{}).
		Where("user_id = ? AND status = ?", userID, groupdomain.MembershipStatusActive).
		Pluck("group_id", &ids).Error
	return ids, err
}

func (r *GormGroupMembershipRepository) CountActive(ctx context.Context, groupID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&groupdomain.Membership{}).
		Where("group_id = ? AND status = ?", groupID, groupdomain.MembershipStatusActive).
		Count(&count).Error
	return count, err
}
