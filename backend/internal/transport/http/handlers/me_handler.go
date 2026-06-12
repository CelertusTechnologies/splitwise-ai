package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
)

type MeHandler struct {
	users repository.UserRepository
}

func NewMeHandler(users repository.UserRepository) *MeHandler {
	return &MeHandler{users: users}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid authenticated user")
		return
	}

	found, err := h.users.FindByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "user_not_found", "User not found.")
		return
	}

	response.OK(c, gin.H{"user": toUserResponse(found)})
}
