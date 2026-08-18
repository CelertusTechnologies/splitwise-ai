package expense

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SplitEqual      = "equal"
	SplitUnequal    = "unequal"
	SplitPercentage = "percentage"
	SplitShares     = "shares"

	StatusActive  = "active"
	StatusDeleted = "deleted"
)

type Expense struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	GroupID         uuid.UUID      `gorm:"column:group_id"`
	CreatedByUserID uuid.UUID      `gorm:"column:created_by_user_id"`
	PaidByUserID    uuid.UUID      `gorm:"column:paid_by_user_id"`
	CategoryID      *uuid.UUID     `gorm:"column:category_id"`
	Title           string         `gorm:"column:title"`
	Description     *string        `gorm:"column:description"`
	AmountMinor     int64          `gorm:"column:amount_minor"`
	Currency        string         `gorm:"column:currency"`
	SplitMethod     string         `gorm:"column:split_method"`
	ExpenseDate     time.Time      `gorm:"column:expense_date"`
	Notes           *string        `gorm:"column:notes"`
	Status          string         `gorm:"column:status"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Expense) TableName() string { return "expenses" }

func (e *Expense) BeforeCreate(*gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

type Share struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	ExpenseID          uuid.UUID `gorm:"column:expense_id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	ShareAmountMinor   int64     `gorm:"column:share_amount_minor"`
	SharePercentageBps *int      `gorm:"column:share_percentage_bps"`
	ShareCount         *int      `gorm:"column:share_count"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (Share) TableName() string { return "expense_shares" }

func (s *Share) BeforeCreate(*gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type Category struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Slug string    `gorm:"column:slug"`
	Name string    `gorm:"column:name"`
	Icon string    `gorm:"column:icon"`
}

func (Category) TableName() string { return "expense_categories" }

func (cat *Category) BeforeCreate(*gorm.DB) error {
	if cat.ID == uuid.Nil {
		cat.ID = uuid.New()
	}
	return nil
}
