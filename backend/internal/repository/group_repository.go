package repository

import (
	"context"
	"errors"

	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupRepository interface {
	Create(ctx context.Context, g *groupdomain.Group) error
	FindByID(ctx context.Context, id uuid.UUID) (*groupdomain.Group, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]groupdomain.Group, error)
}

type GormGroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &GormGroupRepository{db: db}
}

func (r *GormGroupRepository) Create(ctx context.Context, g *groupdomain.Group) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *GormGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*groupdomain.Group, error) {
	var found groupdomain.Group
	err := r.db.WithContext(ctx).
		Where("id = ? AND status <> ?", id, groupdomain.StatusDeleted).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormGroupRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]groupdomain.Group, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var found []groupdomain.Group
	err := r.db.WithContext(ctx).
		Where("id IN ? AND status <> ?", ids, groupdomain.StatusDeleted).
		Order("updated_at DESC").
		Find(&found).Error
	return found, err
}
