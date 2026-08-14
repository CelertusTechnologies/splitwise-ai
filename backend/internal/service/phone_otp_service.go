package service

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/nivra/splitwise-ai/backend/internal/config"
	authdomain "github.com/nivra/splitwise-ai/backend/internal/domain/auth"
	userdomain "github.com/nivra/splitwise-ai/backend/internal/domain/user"
	"github.com/nivra/splitwise-ai/backend/internal/platform/security"
	"github.com/nivra/splitwise-ai/backend/internal/repository"
)

var (
	ErrOTPCooldown         = errors.New("please wait before requesting another code")
	ErrOTPInvalid          = errors.New("invalid or expired code")
	ErrOTPTooManyTries     = errors.New("too many attempts, request a new code")
	ErrPhoneAlreadyExists  = errors.New("phone number already exists")
)

const (
	otpTTL             = 10 * time.Minute
	otpRequestCooldown = 60 * time.Second
	otpMaxAttempts     = 5
	otpCodeLength      = 6
)

// PhoneOTPService implements phone-number login as an alternative to
// email/password: request a code, verify it, and — only for numbers with no
// existing account — collect a name/email to finish creating one. It never
// touches the email/password flow in AuthService; it reuses its token
// issuance so both login paths produce identical session tokens.
type PhoneOTPService struct {
	cfg   config.Config
	users repository.UserRepository
	otps  repository.PhoneOTPRepository
	auth  *AuthService
}

func NewPhoneOTPService(cfg config.Config, users repository.UserRepository, otps repository.PhoneOTPRepository, auth *AuthService) *PhoneOTPService {
	return &PhoneOTPService{cfg: cfg, users: users, otps: otps, auth: auth}
}

type RequestOTPResult struct {
	DevCode string
}

func (s *PhoneOTPService) RequestOTP(ctx context.Context, phoneNumber string) (*RequestOTPResult, error) {
	phone := normalizePhone(phoneNumber)

	latest, err := s.otps.FindLatest(ctx, phone)
	if err == nil {
		if time.Since(latest.CreatedAt) < otpRequestCooldown {
			return nil, ErrOTPCooldown
		}
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	code, err := generateOTPCode()
	if err != nil {
		return nil, err
	}

	if err := s.otps.Create(ctx, &authdomain.PhoneOTP{
		PhoneNumber: phone,
		CodeHash:    security.HashOpaqueToken(code),
		ExpiresAt:   time.Now().UTC().Add(otpTTL),
	}); err != nil {
		return nil, err
	}

	result := &RequestOTPResult{}
	if s.cfg.Env != "production" {
		result.DevCode = code
	}
	return result, nil
}

type VerifyOTPResult struct {
	IsNewUser  bool
	AuthResult *AuthResult
}

func (s *PhoneOTPService) VerifyOTP(ctx context.Context, phoneNumber, code string, client AuthClient) (*VerifyOTPResult, error) {
	phone := normalizePhone(phoneNumber)

	if _, err := s.checkOTP(ctx, phone, code); err != nil {
		return nil, err
	}

	found, err := s.users.FindByPhoneNumber(ctx, phone)
	if errors.Is(err, repository.ErrNotFound) {
		// Leave the code unconsumed: CompleteSignup re-checks it once the
		// caller supplies the rest of the profile.
		return &VerifyOTPResult{IsNewUser: true}, nil
	}
	if err != nil {
		return nil, err
	}
	if found.IsSuspended() {
		return nil, ErrAccountDisabled
	}

	if err := s.consumeOTP(ctx, phone); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.users.UpdateLastLogin(ctx, found.ID, now); err != nil {
		return nil, err
	}

	tokens, err := s.auth.issueTokenPair(ctx, *found, client)
	if err != nil {
		return nil, err
	}

	return &VerifyOTPResult{IsNewUser: false, AuthResult: &AuthResult{User: found, Tokens: tokens}}, nil
}

type CompleteSignupInput struct {
	PhoneNumber string
	Code        string
	FullName    string
	Email       string
}

func (s *PhoneOTPService) CompleteSignup(ctx context.Context, input CompleteSignupInput, client AuthClient) (*AuthResult, error) {
	phone := normalizePhone(input.PhoneNumber)

	if _, err := s.checkOTP(ctx, phone, input.Code); err != nil {
		return nil, err
	}

	if _, err := s.users.FindByPhoneNumber(ctx, phone); err == nil {
		return nil, ErrPhoneAlreadyExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	email := normalizeEmail(input.Email)
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// These users only ever authenticate via a fresh OTP, never a password,
	// so the password field is filled with an unguessable, unused secret
	// (recoverable only through the normal email-based forgot-password flow).
	throwaway, _, err := security.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	passwordHash, err := security.HashPassword(throwaway)
	if err != nil {
		return nil, err
	}

	newUser := &userdomain.User{
		FullName:          strings.TrimSpace(input.FullName),
		Email:             email,
		PhoneNumber:       &phone,
		PasswordHash:      passwordHash,
		PreferredCurrency: "INR",
		ThemePreference:   "system",
		Role:              userdomain.RoleUser,
		Status:            userdomain.StatusActive, // phone ownership already proven via OTP
	}
	if err := s.users.Create(ctx, newUser); err != nil {
		return nil, err
	}

	if err := s.consumeOTP(ctx, phone); err != nil {
		return nil, err
	}

	tokens, err := s.auth.issueTokenPair(ctx, *newUser, client)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: newUser, Tokens: tokens}, nil
}

// checkOTP validates, without consuming, that code is the current, unexpired,
// not-yet-consumed code for phone, tracking failed attempts as it goes. A
// successful check can safely be repeated (e.g. VerifyOTP followed later by
// CompleteSignup for a brand new number) since it never marks the code used.
func (s *PhoneOTPService) checkOTP(ctx context.Context, phone, code string) (*authdomain.PhoneOTP, error) {
	latest, err := s.otps.FindLatest(ctx, phone)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrOTPInvalid
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if latest.ConsumedAt != nil || now.After(latest.ExpiresAt) {
		return nil, ErrOTPInvalid
	}
	if latest.AttemptCount >= otpMaxAttempts {
		return nil, ErrOTPTooManyTries
	}
	if latest.CodeHash != security.HashOpaqueToken(strings.TrimSpace(code)) {
		_ = s.otps.IncrementAttempt(ctx, latest.ID)
		return nil, ErrOTPInvalid
	}

	return latest, nil
}

func (s *PhoneOTPService) consumeOTP(ctx context.Context, phone string) error {
	latest, err := s.otps.FindLatest(ctx, phone)
	if err != nil {
		return err
	}
	return s.otps.MarkConsumed(ctx, latest.ID, time.Now().UTC())
}

func generateOTPCode() (string, error) {
	const digits = "0123456789"
	buf := make([]byte, otpCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, otpCodeLength)
	for i, b := range buf {
		code[i] = digits[int(b)%len(digits)]
	}
	return string(code), nil
}

func normalizePhone(raw string) string {
	return strings.TrimSpace(raw)
}
