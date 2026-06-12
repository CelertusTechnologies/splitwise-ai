package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nivra/splitwise-ai/backend/internal/config"
	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	userdomain "github.com/nivra/splitwise-ai/backend/internal/domain/user"
	"github.com/nivra/splitwise-ai/backend/internal/platform/security"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrInvalidAuthToken   = errors.New("invalid auth token")
)

type AuthService struct {
	cfg           config.Config
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	oneTimeTokens repository.OneTimeTokenRepository
	jwtManager    *security.JWTManager
}

type AuthClient struct {
	IPAddress         string
	UserAgent         string
	DeviceFingerprint string
}

type SignUpInput struct {
	FullName          string
	Email             string
	PhoneNumber       *string
	Password          string
	PreferredCurrency string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type AuthResult struct {
	User                      *userdomain.User
	Tokens                    TokenPair
	DevEmailVerificationToken string
}

func NewAuthService(
	cfg config.Config,
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	oneTimeTokens repository.OneTimeTokenRepository,
	jwtManager *security.JWTManager,
) *AuthService {
	return &AuthService{
		cfg:           cfg,
		users:         users,
		refreshTokens: refreshTokens,
		oneTimeTokens: oneTimeTokens,
		jwtManager:    jwtManager,
	}
}

func (s *AuthService) SignUp(ctx context.Context, input SignUpInput, client AuthClient) (*AuthResult, error) {
	email := normalizeEmail(input.Email)
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	currency := strings.ToUpper(strings.TrimSpace(input.PreferredCurrency))
	if currency == "" {
		currency = "INR"
	}

	newUser := &userdomain.User{
		FullName:          strings.TrimSpace(input.FullName),
		Email:             email,
		PhoneNumber:       normalizeOptional(input.PhoneNumber),
		PasswordHash:      passwordHash,
		PreferredCurrency: currency,
		ThemePreference:   "system",
		Role:              userdomain.RoleUser,
		Status:            userdomain.StatusPendingVerification,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		return nil, err
	}

	emailToken, emailTokenHash, err := security.NewOpaqueToken()
	if err != nil {
		return nil, err
	}

	if err := s.oneTimeTokens.Create(ctx, &authdomain.OneTimeToken{
		UserID:    newUser.ID,
		Purpose:   authdomain.OneTimePurposeEmailVerification,
		TokenHash: emailTokenHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}); err != nil {
		return nil, err
	}

	tokens, err := s.issueTokenPair(ctx, *newUser, client)
	if err != nil {
		return nil, err
	}

	result := &AuthResult{
		User:   newUser,
		Tokens: tokens,
	}
	if s.cfg.Env != "production" {
		result.DevEmailVerificationToken = emailToken
	}

	return result, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput, client AuthClient) (*AuthResult, error) {
	found, err := s.users.FindByEmail(ctx, normalizeEmail(input.Email))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if found.IsSuspended() {
		return nil, ErrAccountDisabled
	}
	if err := security.ComparePassword(found.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	if err := s.users.UpdateLastLogin(ctx, found.ID, now); err != nil {
		return nil, err
	}
	found.LastLoginAt = &now

	tokens, err := s.issueTokenPair(ctx, *found, client)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: found, Tokens: tokens}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string, client AuthClient) (*AuthResult, error) {
	claims, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidAuthToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrInvalidAuthToken
	}

	now := time.Now().UTC()
	storedToken, err := s.refreshTokens.FindActiveByTokenID(ctx, claims.ID, now)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidAuthToken
	}
	if err != nil {
		return nil, err
	}
	if storedToken.UserID != userID {
		return nil, ErrInvalidAuthToken
	}

	foundUser, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if foundUser.IsSuspended() {
		return nil, ErrAccountDisabled
	}

	nextRefreshTokenID := uuid.NewString()
	access, err := s.jwtManager.GenerateAccessToken(foundUser.ID, foundUser.Role)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwtManager.GenerateRefreshToken(foundUser.ID, foundUser.Role, nextRefreshTokenID)
	if err != nil {
		return nil, err
	}

	if err := s.refreshTokens.Create(ctx, &authdomain.RefreshToken{
		UserID:            foundUser.ID,
		TokenID:           refresh.TokenID,
		DeviceFingerprint: optionalString(client.DeviceFingerprint),
		IPAddress:         optionalString(client.IPAddress),
		UserAgent:         optionalString(client.UserAgent),
		ExpiresAt:         refresh.ExpiresAt,
	}); err != nil {
		return nil, err
	}

	if err := s.refreshTokens.RevokeByTokenID(ctx, claims.ID, &nextRefreshTokenID, now); err != nil {
		return nil, err
	}

	return &AuthResult{
		User: foundUser,
		Tokens: TokenPair{
			AccessToken:           access.Token,
			RefreshToken:          refresh.Token,
			AccessTokenExpiresAt:  access.ExpiresAt,
			RefreshTokenExpiresAt: refresh.ExpiresAt,
		},
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidAuthToken
	}
	return s.refreshTokens.RevokeByTokenID(ctx, claims.ID, nil, time.Now().UTC())
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) (string, error) {
	found, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, repository.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	plain, tokenHash, err := security.NewOpaqueToken()
	if err != nil {
		return "", err
	}

	if err := s.oneTimeTokens.Create(ctx, &authdomain.OneTimeToken{
		UserID:    found.ID,
		Purpose:   authdomain.OneTimePurposePasswordReset,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	}); err != nil {
		return "", err
	}

	if s.cfg.Env == "production" {
		return "", nil
	}
	return plain, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	now := time.Now().UTC()
	tokenHash := security.HashOpaqueToken(token)

	foundToken, err := s.oneTimeTokens.FindActiveByHash(ctx, authdomain.OneTimePurposePasswordReset, tokenHash, now)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrInvalidAuthToken
	}
	if err != nil {
		return err
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.users.UpdatePassword(ctx, foundToken.UserID, passwordHash); err != nil {
		return err
	}
	if err := s.oneTimeTokens.MarkUsed(ctx, tokenHash, now); err != nil {
		return err
	}
	return s.refreshTokens.RevokeAllForUser(ctx, foundToken.UserID, now)
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	now := time.Now().UTC()
	tokenHash := security.HashOpaqueToken(token)

	foundToken, err := s.oneTimeTokens.FindActiveByHash(ctx, authdomain.OneTimePurposeEmailVerification, tokenHash, now)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrInvalidAuthToken
	}
	if err != nil {
		return err
	}

	if err := s.users.MarkEmailVerified(ctx, foundToken.UserID, now); err != nil {
		return err
	}
	return s.oneTimeTokens.MarkUsed(ctx, tokenHash, now)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user userdomain.User, client AuthClient) (TokenPair, error) {
	refreshTokenID := uuid.NewString()

	access, err := s.jwtManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.jwtManager.GenerateRefreshToken(user.ID, user.Role, refreshTokenID)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.refreshTokens.Create(ctx, &authdomain.RefreshToken{
		UserID:            user.ID,
		TokenID:           refresh.TokenID,
		DeviceFingerprint: optionalString(client.DeviceFingerprint),
		IPAddress:         optionalString(client.IPAddress),
		UserAgent:         optionalString(client.UserAgent),
		ExpiresAt:         refresh.ExpiresAt,
	}); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:           access.Token,
		RefreshToken:          refresh.Token,
		AccessTokenExpiresAt:  access.ExpiresAt,
		RefreshTokenExpiresAt: refresh.ExpiresAt,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
