package repository

import (
	"context"

	expensedomain "github.com/nivra/splitwise-ai/backend/internal/domain/expense"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExpenseShareRepository interface {
	CreateBatch(ctx context.Context, shares []expensedomain.Share) error
	ListByExpense(ctx context.Context, expenseID uuid.UUID) ([]expensedomain.Share, error)
	SumShareByUser(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error)
}

type GormExpenseShareRepository struct {
	db *gorm.DB
}

func NewExpenseShareRepository(db *gorm.DB) ExpenseShareRepository {
	return &GormExpenseShareRepository{db: db}
}

func (r *GormExpenseShareRepository) CreateBatch(ctx context.Context, shares []expensedomain.Share) error {
	if len(shares) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&shares).Error
}

func (r *GormExpenseShareRepository) ListByExpense(ctx context.Context, expenseID uuid.UUID) ([]expensedomain.Share, error) {
	var found []expensedomain.Share
	err := r.db.WithContext(ctx).Where("expense_id = ?", expenseID).Find(&found).Error
	return found, err
}

func (r *GormExpenseShareRepository) SumShareByUser(ctx context.Context, groupID uuid.UUID) (map[uuid.UUID]int64, error) {
	type row struct {
		UserID uuid.UUID
		Total  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("expense_shares").
		Select("expense_shares.user_id, SUM(expense_shares.share_amount_minor) as total").
		Joins("JOIN expenses ON expenses.id = expense_shares.expense_id").
		Where("expenses.group_id = ? AND expenses.status = ?", groupID, expensedomain.StatusActive).
		Group("expense_shares.user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]int64, len(rows))
	for _, r := range rows {
		result[r.UserID] = r.Total
	}
	return result, nil
}
