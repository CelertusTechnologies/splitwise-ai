package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	userdomain "github.com/nivra/splitwise-ai/backend/internal/domain/user"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type signUpRequest struct {
	FullName          string  `json:"full_name" binding:"required,min=2,max=120"`
	Email             string  `json:"email" binding:"required,email,max=180"`
	PhoneNumber       *string `json:"phone_number" binding:"omitempty,min=8,max=20"`
	Password          string  `json:"password" binding:"required,min=12,max=128"`
	PreferredCurrency string  `json:"preferred_currency" binding:"omitempty,len=3"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=180"`
	Password string `json:"password" binding:"required,min=12,max=128"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=180"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=12,max=128"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type userResponse struct {
	ID                string  `json:"id"`
	FullName          string  `json:"full_name"`
	Email             string  `json:"email"`
	PhoneNumber       *string `json:"phone_number"`
	ProfilePictureURL *string `json:"profile_picture_url"`
	PreferredCurrency string  `json:"preferred_currency"`
	ThemePreference   string  `json:"theme_preference"`
	Role              string  `json:"role"`
	Status            string  `json:"status"`
	EmailVerified     bool    `json:"email_verified"`
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req signUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.authService.SignUp(c.Request.Context(), service.SignUpInput{
		FullName:          req.FullName,
		Email:             req.Email,
		PhoneNumber:       req.PhoneNumber,
		Password:          req.Password,
		PreferredCurrency: req.PreferredCurrency,
	}, clientMeta(c))
	if err != nil {
		writeAuthError(c, err)
		return
	}

	payload := gin.H{
		"user":   toUserResponse(result.User),
		"tokens": result.Tokens,
	}
	if result.DevEmailVerificationToken != "" {
		payload["dev_email_verification_token"] = result.DevEmailVerificationToken
	}

	response.Created(c, payload)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, clientMeta(c))
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, gin.H{
		"user":   toUserResponse(result.User),
		"tokens": result.Tokens,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken, clientMeta(c))
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, gin.H{
		"user":   toUserResponse(result.User),
		"tokens": result.Tokens,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeAuthError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	devToken, err := h.authService.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	payload := gin.H{"message": "If the email exists, a reset link will be sent."}
	if devToken != "" {
		payload["dev_reset_token"] = devToken
	}

	response.Accepted(c, payload)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, gin.H{"message": "Password reset successfully."})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, gin.H{"message": "Email verified successfully."})
}

func clientMeta(c *gin.Context) service.AuthClient {
	return service.AuthClient{
		IPAddress:         c.ClientIP(),
		UserAgent:         c.GetHeader("User-Agent"),
		DeviceFingerprint: c.GetHeader("X-Device-Fingerprint"),
	}
}

func toUserResponse(user *userdomain.User) userResponse {
	return userResponse{
		ID:                user.ID.String(),
		FullName:          user.FullName,
		Email:             user.Email,
		PhoneNumber:       user.PhoneNumber,
		ProfilePictureURL: user.ProfilePictureURL,
		PreferredCurrency: user.PreferredCurrency,
		ThemePreference:   user.ThemePreference,
		Role:              user.Role,
		Status:            user.Status,
		EmailVerified:     user.EmailVerifiedAt != nil,
	}
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyExists):
		response.Error(c, http.StatusConflict, "email_already_exists", "An account with this email already exists.")
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
	case errors.Is(err, service.ErrInvalidAuthToken):
		response.Error(c, http.StatusUnauthorized, "invalid_token", "The token is invalid or expired.")
	case errors.Is(err, service.ErrAccountDisabled):
		response.Error(c, http.StatusForbidden, "account_disabled", "This account is not active.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
