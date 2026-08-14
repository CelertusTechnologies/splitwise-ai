package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OneTimePurposeEmailVerification = "email_verification"
	OneTimePurposePasswordReset     = "password_reset"
)

type RefreshToken struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID  `gorm:"column:user_id"`
	TokenID           string     `gorm:"column:token_id"`
	ReplacedByTokenID *string    `gorm:"column:replaced_by_token_id"`
	DeviceFingerprint *string    `gorm:"column:device_fingerprint"`
	IPAddress         *string    `gorm:"column:ip_address"`
	UserAgent         *string    `gorm:"column:user_agent"`
	ExpiresAt         time.Time  `gorm:"column:expires_at"`
	RevokedAt         *time.Time `gorm:"column:revoked_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// BeforeCreate assigns a portable UUID primary key (see User.BeforeCreate).
func (t *RefreshToken) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type OneTimeToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"column:user_id"`
	Purpose   string     `gorm:"column:purpose"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (OneTimeToken) TableName() string {
	return "one_time_tokens"
}

// BeforeCreate assigns a portable UUID primary key (see User.BeforeCreate).
func (t *OneTimeToken) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// PhoneOTP is a short-lived numeric code sent to a phone number to log in or,
// for a number with no existing account, to prove ownership before the
// caller supplies the rest of the signup details.
type PhoneOTP struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PhoneNumber  string     `gorm:"column:phone_number"`
	CodeHash     string     `gorm:"column:code_hash"`
	AttemptCount int        `gorm:"column:attempt_count"`
	ExpiresAt    time.Time  `gorm:"column:expires_at"`
	ConsumedAt   *time.Time `gorm:"column:consumed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (PhoneOTP) TableName() string {
	return "phone_otp_codes"
}

// BeforeCreate assigns a portable UUID primary key (see User.BeforeCreate).
func (t *PhoneOTP) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
