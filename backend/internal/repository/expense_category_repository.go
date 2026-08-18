package repository

import (
	"context"
	"errors"

	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
	"gorm.io/gorm"
)

type ExpenseCategoryRepository interface {
	List(ctx context.Context) ([]expensedomain.Category, error)
	FindBySlug(ctx context.Context, slug string) (*expensedomain.Category, error)
}

type GormExpenseCategoryRepository struct {
	db *gorm.DB
}

func NewExpenseCategoryRepository(db *gorm.DB) ExpenseCategoryRepository {
	return &GormExpenseCategoryRepository{db: db}
}

func (r *GormExpenseCategoryRepository) List(ctx context.Context) ([]expensedomain.Category, error) {
	var found []expensedomain.Category
	err := r.db.WithContext(ctx).Order("name").Find(&found).Error
	return found, err
}

func (r *GormExpenseCategoryRepository) FindBySlug(ctx context.Context, slug string) (*expensedomain.Category, error) {
	var found expensedomain.Category
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}
