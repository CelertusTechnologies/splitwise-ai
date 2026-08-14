package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nivra/splitwise-ai/backend/internal/service"
	"github.com/nivra/splitwise-ai/backend/internal/transport/http/response"
)

type PhoneOTPHandler struct {
	otps *service.PhoneOTPService
}

func NewPhoneOTPHandler(otps *service.PhoneOTPService) *PhoneOTPHandler {
	return &PhoneOTPHandler{otps: otps}
}

type requestOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required,min=8,max=20"`
}

type verifyOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required,min=8,max=20"`
	Code        string `json:"code" binding:"required,len=6,numeric"`
}

type completeOTPSignupRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required,min=8,max=20"`
	Code        string `json:"code" binding:"required,len=6,numeric"`
	FullName    string `json:"full_name" binding:"required,min=2,max=120"`
	Email       string `json:"email" binding:"required,email,max=180"`
}

func (h *PhoneOTPHandler) Request(c *gin.Context) {
	var req requestOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.otps.RequestOTP(c.Request.Context(), req.PhoneNumber)
	if err != nil {
		writeOTPError(c, err)
		return
	}

	payload := gin.H{"message": "A code has been sent to that number."}
	if result.DevCode != "" {
		payload["dev_otp"] = result.DevCode
	}
	response.OK(c, payload)
}

func (h *PhoneOTPHandler) Verify(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.otps.VerifyOTP(c.Request.Context(), req.PhoneNumber, req.Code, clientMeta(c))
	if err != nil {
		writeOTPError(c, err)
		return
	}

	if result.IsNewUser {
		response.OK(c, gin.H{"is_new_user": true})
		return
	}

	response.OK(c, gin.H{
		"is_new_user": false,
		"user":        toUserResponse(result.AuthResult.User),
		"tokens":      result.AuthResult.Tokens,
	})
}

func (h *PhoneOTPHandler) CompleteSignup(c *gin.Context) {
	var req completeOTPSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	result, err := h.otps.CompleteSignup(c.Request.Context(), service.CompleteSignupInput{
		PhoneNumber: req.PhoneNumber,
		Code:        req.Code,
		FullName:    req.FullName,
		Email:       req.Email,
	}, clientMeta(c))
	if err != nil {
		writeOTPError(c, err)
		return
	}

	response.Created(c, gin.H{
		"user":   toUserResponse(result.User),
		"tokens": result.Tokens,
	})
}

func writeOTPError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOTPCooldown):
		response.Error(c, http.StatusTooManyRequests, "otp_cooldown", "Please wait a moment before requesting another code.")
	case errors.Is(err, service.ErrOTPInvalid):
		response.Error(c, http.StatusBadRequest, "otp_invalid", "That code is invalid or has expired.")
	case errors.Is(err, service.ErrOTPTooManyTries):
		response.Error(c, http.StatusTooManyRequests, "otp_too_many_tries", "Too many attempts. Request a new code.")
	case errors.Is(err, service.ErrPhoneAlreadyExists):
		response.Error(c, http.StatusConflict, "phone_already_exists", "An account with this phone number already exists.")
	case errors.Is(err, service.ErrEmailAlreadyExists):
		response.Error(c, http.StatusConflict, "email_already_exists", "An account with this email already exists.")
	case errors.Is(err, service.ErrAccountDisabled):
		response.Error(c, http.StatusForbidden, "account_disabled", "This account has been disabled.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
