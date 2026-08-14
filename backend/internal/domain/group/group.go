package group

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TypeTrip      = "trip"
	TypeFamily    = "family"
	TypeFriends   = "friends"
	TypeFlatmates = "flatmates"
	TypeCouple    = "couple"
	TypeCustom    = "custom"

	StatusActive   = "active"
	StatusArchived = "archived"
	StatusDeleted  = "deleted"

	InvitePolicyOwners  = "owners"
	InvitePolicyAdmins  = "admins"
	InvitePolicyMembers = "members"

	MembershipRoleOwner  = "owner"
	MembershipRoleAdmin  = "admin"
	MembershipRoleMember = "member"

	MembershipStatusInvited = "invited"
	MembershipStatusActive  = "active"
	MembershipStatusLeft    = "left"
	MembershipStatusRemoved = "removed"
)

type Group struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	OwnerUserID     uuid.UUID `gorm:"column:owner_user_id"`
	Name            string    `gorm:"column:name"`
	Description     *string   `gorm:"column:description"`
	GroupType       string    `gorm:"column:group_type"`
	PhotoURL        *string   `gorm:"column:photo_url"`
	DefaultCurrency string    `gorm:"column:default_currency"`
	InvitePolicy    string    `gorm:"column:invite_policy"`
	Status          string    `gorm:"column:status"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (Group) TableName() string { return "groups" }

func (g *Group) BeforeCreate(*gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type Membership struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GroupID   uuid.UUID  `gorm:"column:group_id"`
	UserID    uuid.UUID  `gorm:"column:user_id"`
	Role      string     `gorm:"column:role"`
	Status    string     `gorm:"column:status"`
	JoinedAt  *time.Time `gorm:"column:joined_at"`
	LeftAt    *time.Time `gorm:"column:left_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (Membership) TableName() string { return "group_memberships" }

func (m *Membership) BeforeCreate(*gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (m Membership) IsActive() bool {
	return m.Status == MembershipStatusActive
}

func (m Membership) CanInvite(policy string) bool {
	switch policy {
	case InvitePolicyMembers:
		return true
	case InvitePolicyAdmins:
		return m.Role == MembershipRoleOwner || m.Role == MembershipRoleAdmin
	default: // InvitePolicyOwners
		return m.Role == MembershipRoleOwner
	}
}

type Invite struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GroupID         uuid.UUID  `gorm:"column:group_id"`
	CreatedByUserID uuid.UUID  `gorm:"column:created_by_user_id"`
	InviteCode      string     `gorm:"column:invite_code"`
	MaxUses         *int       `gorm:"column:max_uses"`
	UsedCount       int        `gorm:"column:used_count"`
	ExpiresAt       time.Time  `gorm:"column:expires_at"`
	RevokedAt       *time.Time `gorm:"column:revoked_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (Invite) TableName() string { return "group_invites" }

func (i *Invite) BeforeCreate(*gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (i Invite) IsUsable(now time.Time) bool {
	if i.RevokedAt != nil {
		return false
	}
	if now.After(i.ExpiresAt) {
		return false
	}
	if i.MaxUses != nil && i.UsedCount >= *i.MaxUses {
		return false
	}
	return true
}
