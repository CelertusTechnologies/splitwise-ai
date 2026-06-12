package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	StatusActive              = "active"
	StatusPendingVerification = "pending_verification"
	StatusSuspended           = "suspended"
	StatusDeleted             = "deleted"
)

type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey"`
	FullName          string         `gorm:"column:full_name"`
	Email             string         `gorm:"column:email"`
	PhoneNumber       *string        `gorm:"column:phone_number"`
	PasswordHash      string         `gorm:"column:password_hash"`
	ProfilePictureURL *string        `gorm:"column:profile_picture_url"`
	PreferredCurrency string         `gorm:"column:preferred_currency"`
	ThemePreference   string         `gorm:"column:theme_preference"`
	Role              string         `gorm:"column:role"`
	Status            string         `gorm:"column:status"`
	EmailVerifiedAt   *time.Time     `gorm:"column:email_verified_at"`
	LastLoginAt       *time.Time     `gorm:"column:last_login_at"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (User) TableName() string {
	return "users"
}

// BeforeCreate assigns a UUID primary key in application code so the model is
// portable across databases (Postgres' gen_random_uuid() default is unavailable
// on SQLite). Postgres keeps its column-level default for non-ORM inserts.
func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u User) IsSuspended() bool {
	return u.Status == StatusSuspended || u.Status == StatusDeleted
}
