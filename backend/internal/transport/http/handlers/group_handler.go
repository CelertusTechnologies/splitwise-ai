package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	groupdomain "github.com/nivra/splitwise-ai/backend/internal/domain/group"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
)

type GroupHandler struct {
	groups      *service.GroupService
	frontendURL string
}

func NewGroupHandler(groups *service.GroupService, frontendURL string) *GroupHandler {
	return &GroupHandler{groups: groups, frontendURL: frontendURL}
}

type createGroupRequest struct {
	Name            string  `json:"name" binding:"required,min=2,max=120"`
	Description     *string `json:"description" binding:"omitempty,max=500"`
	GroupType       string  `json:"group_type" binding:"omitempty,oneof=trip family friends flatmates couple custom"`
	DefaultCurrency string  `json:"default_currency" binding:"omitempty,len=3"`
}

type joinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

type groupResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	GroupType       string    `json:"group_type"`
	DefaultCurrency string    `json:"default_currency"`
	Status          string    `json:"status"`
	MyRole          string    `json:"my_role,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func toGroupResponse(g groupdomain.Group, role string) groupResponse {
	return groupResponse{
		ID:              g.ID.String(),
		Name:            g.Name,
		Description:     g.Description,
		GroupType:       g.GroupType,
		DefaultCurrency: g.DefaultCurrency,
		Status:          g.Status,
		MyRole:          role,
		CreatedAt:       g.CreatedAt,
	}
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid authenticated user")
		return uuid.Nil, false
	}
	return userID, true
}

func (h *GroupHandler) Create(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	created, err := h.groups.CreateGroup(c.Request.Context(), userID, service.CreateGroupInput{
		Name:            req.Name,
		Description:     req.Description,
		GroupType:       req.GroupType,
		DefaultCurrency: req.DefaultCurrency,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "Could not create group.")
		return
	}

	response.Created(c, gin.H{"group": toGroupResponse(*created, groupdomain.MembershipRoleOwner)})
}

func (h *GroupHandler) List(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	found, err := h.groups.ListMyGroups(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "Could not load groups.")
		return
	}

	items := make([]groupResponse, 0, len(found))
	for _, g := range found {
		items = append(items, toGroupResponse(g, ""))
	}

	response.OK(c, gin.H{"groups": items})
}

func (h *GroupHandler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	found, membership, err := h.groups.GetGroup(c.Request.Context(), userID, groupID)
	if writeGroupServiceError(c, err) {
		return
	}

	response.OK(c, gin.H{"group": toGroupResponse(*found, membership.Role)})
}

type memberResponse struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func (h *GroupHandler) ListMembers(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	found, err := h.groups.ListMembers(c.Request.Context(), userID, groupID)
	if writeGroupServiceError(c, err) {
		return
	}

	items := make([]memberResponse, 0, len(found))
	for _, m := range found {
		items = append(items, memberResponse{UserID: m.UserID.String(), FullName: m.FullName, Email: m.Email, Role: m.Role})
	}
	response.OK(c, gin.H{"members": items})
}

func (h *GroupHandler) CreateInvite(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", "invalid group id")
		return
	}

	invite, err := h.groups.CreateInvite(c.Request.Context(), userID, groupID)
	if writeGroupServiceError(c, err) {
		return
	}

	inviteURL := strings.TrimRight(h.frontendURL, "/") + "/join/" + invite.InviteCode
	response.Created(c, gin.H{"invite": gin.H{
		"invite_code": invite.InviteCode,
		"invite_url":  inviteURL,
		"expires_at":  invite.ExpiresAt,
	}})
}

func (h *GroupHandler) Join(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req joinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	found, err := h.groups.JoinByInviteCode(c.Request.Context(), userID, req.InviteCode)
	if writeGroupServiceError(c, err) {
		return
	}

	response.OK(c, gin.H{"group": toGroupResponse(*found, groupdomain.MembershipRoleMember)})
}

// writeGroupServiceError maps a group service error to an HTTP response and
// reports whether it handled (i.e. the caller should stop processing).
func writeGroupServiceError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, service.ErrGroupNotFound), errors.Is(err, service.ErrNotAMember):
		response.Error(c, http.StatusNotFound, "group_not_found", "Group not found.")
	case errors.Is(err, service.ErrForbidden):
		response.Error(c, http.StatusForbidden, "forbidden", "You don't have permission to do that.")
	case errors.Is(err, service.ErrInviteInvalid):
		response.Error(c, http.StatusBadRequest, "invite_invalid", "This invite link is invalid, expired, or has been used up.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
	return true
}
