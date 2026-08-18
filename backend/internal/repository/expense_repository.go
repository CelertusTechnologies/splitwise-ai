package repository

import (
	"context"
	"errors"
	"time"

	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExpenseRepository interface {
	Create(ctx context.Context, e *expensedomain.Expense) error
	FindByID(ctx context.Context, id uuid.UUID) (*expensedomain.Expense, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]expensedomain.Expense, error)
	SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) error
	SumPaidByUser(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error)
}

type GormExpenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &GormExpenseRepository{db: db}
}

func (r *GormExpenseRepository) Create(ctx context.Context, e *expensedomain.Expense) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *GormExpenseRepository) FindByID(ctx context.Context, id uuid.UUID) (*expensedomain.Expense, error) {
	var found expensedomain.Expense
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, expensedomain.StatusActive).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &found, err
}

func (r *GormExpenseRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]expensedomain.Expense, error) {
	var found []expensedomain.Expense
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = ?", groupID, expensedomain.StatusActive).
		Order("expense_date DESC, created_at DESC").
		Find(&found).Error
	return found, err
}

func (r *GormExpenseRepository) SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&expensedomain.Expense{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     expensedomain.StatusDeleted,
			"deleted_at": at,
			"updated_at": at,
		}).Error
}

func (r *GormExpenseRepository) SumPaidByUser(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error) {
	type row struct {
		PaidByUserID uuid.UUID
		Total        int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&expensedomain.Expense{}).
		Select("paid_by_user_id, SUM(amount_minor) as total").
		Where("group_id = ? AND status = ?", groupID, expensedomain.StatusActive).
		Group("paid_by_user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]int64, len(rows))
	for _, r := range rows {
		result[r.PaidByUserID] = r.Total
	}
	return result, nil
}
