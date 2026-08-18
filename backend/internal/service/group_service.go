package service

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrGroupNotFound  = errors.New("group not found")
	ErrNotAMember     = errors.New("not a member of this group")
	ErrForbidden      = errors.New("not allowed to perform this action")
	ErrInviteInvalid  = errors.New("invite code is invalid, expired, or has been used up")
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars (0/O, 1/I/L)

type GroupService struct {
	groups      repository.GroupRepository
	memberships repository.GroupMembershipRepository
	invites     repository.GroupInviteRepository
	users       repository.UserRepository
}

func NewGroupService(
	groups repository.GroupRepository,
	memberships repository.GroupMembershipRepository,
	invites repository.GroupInviteRepository,
	users repository.UserRepository,
) *GroupService {
	return &GroupService{groups: groups, memberships: memberships, invites: invites, users: users}
}

type MemberInfo struct {
	UserID   uuid.UUID
	FullName string
	Email    string
	Role     string
}

type CreateGroupInput struct {
	Name            string
	Description     *string
	GroupType       string
	DefaultCurrency string
}

type GroupWithRole struct {
	Group  groupdomain.Group
	Role   string
	Status string
}

func (s *GroupService) CreateGroup(ctx context.Context, ownerID uuid.UUID, input CreateGroupInput) (*groupdomain.Group, error) {
	groupType := strings.TrimSpace(input.GroupType)
	if groupType == "" {
		groupType = groupdomain.TypeCustom
	}

	currency := strings.ToUpper(strings.TrimSpace(input.DefaultCurrency))
	if currency == "" {
		currency = "INR"
	}

	newGroup := &groupdomain.Group{
		OwnerUserID:     ownerID,
		Name:            strings.TrimSpace(input.Name),
		Description:     input.Description,
		GroupType:       groupType,
		DefaultCurrency: currency,
		InvitePolicy:    groupdomain.InvitePolicyAdmins,
		Status:          groupdomain.StatusActive,
	}

	if err := s.groups.Create(ctx, newGroup); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	membership := &groupdomain.Membership{
		GroupID:  newGroup.ID,
		UserID:   ownerID,
		Role:     groupdomain.MembershipRoleOwner,
		Status:   groupdomain.MembershipStatusActive,
		JoinedAt: &now,
	}
	if err := s.memberships.Create(ctx, membership); err != nil {
		return nil, err
	}

	return newGroup, nil
}

func (s *GroupService) ListMyGroups(ctx context.Context, userID uuid.UUID) ([]groupdomain.Group, error) {
	ids, err := s.memberships.ListActiveGroupIDsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.groups.ListByIDs(ctx, ids)
}

func (s *GroupService) GetGroup(ctx context.Context, userID, groupID uuid.UUID) (*groupdomain.Group, *groupdomain.Membership, error) {
	membership, err := s.memberships.FindActive(ctx, groupID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrNotAMember
	}
	if err != nil {
		return nil, nil, err
	}

	found, err := s.groups.FindByID(ctx, groupID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	return found, membership, nil
}

func (s *GroupService) ListMembers(ctx context.Context, requesterID, groupID uuid.UUID) ([]MemberInfo, error) {
	if _, err := s.memberships.FindActive(ctx, groupID, requesterID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotAMember
		}
		return nil, err
	}

	memberships, err := s.memberships.ListActiveByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	members := make([]MemberInfo, 0, len(memberships))
	for _, m := range memberships {
		user, err := s.users.FindByID(ctx, m.UserID)
		if errors.Is(err, repository.ErrNotFound) {
			continue // shouldn't happen, but don't fail the whole list over one stale row
		}
		if err != nil {
			return nil, err
		}
		members = append(members, MemberInfo{UserID: user.ID, FullName: user.FullName, Email: user.Email, Role: m.Role})
	}
	return members, nil
}

func (s *GroupService) CreateInvite(ctx context.Context, userID, groupID uuid.UUID) (*groupdomain.Invite, error) {
	found, membership, err := s.GetGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !membership.CanInvite(found.InvitePolicy) {
		return nil, ErrForbidden
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invite := &groupdomain.Invite{
		GroupID:         groupID,
		CreatedByUserID: userID,
		InviteCode:      code,
		ExpiresAt:       time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

func (s *GroupService) JoinByInviteCode(ctx context.Context, userID uuid.UUID, code string) (*groupdomain.Group, error) {
	invite, err := s.invites.FindByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInviteInvalid
	}
	if err != nil {
		return nil, err
	}
	if !invite.IsUsable(time.Now().UTC()) {
		return nil, ErrInviteInvalid
	}

	found, err := s.groups.FindByID(ctx, invite.GroupID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	if existing, err := s.memberships.FindActive(ctx, invite.GroupID, userID); err == nil && existing != nil {
		return found, nil // already a member; joining again is a harmless no-op
	}

	now := time.Now().UTC()
	membership := &groupdomain.Membership{
		GroupID:  invite.GroupID,
		UserID:   userID,
		Role:     groupdomain.MembershipRoleMember,
		Status:   groupdomain.MembershipStatusActive,
		JoinedAt: &now,
	}
	if err := s.memberships.Create(ctx, membership); err != nil {
		return nil, err
	}

	if err := s.invites.IncrementUse(ctx, invite.ID); err != nil {
		return nil, err
	}

	return found, nil
}

func generateInviteCode() (string, error) {
	const length = 8
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, length)
	for i, b := range buf {
		code[i] = inviteCodeAlphabet[int(b)%len(inviteCodeAlphabet)]
	}
	return string(code), nil
}
