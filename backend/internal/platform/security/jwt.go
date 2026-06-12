package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nivra/splitwise-ai/backend/internal/config"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTManager struct {
	issuer        string
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type Claims struct {
	UserID    string `json:"uid"`
	Role      string `json:"role"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

type SignedToken struct {
	Token     string
	TokenID   string
	ExpiresAt time.Time
}

func NewJWTManager(cfg config.Config) *JWTManager {
	return &JWTManager{
		issuer:        cfg.AppName,
		accessSecret:  []byte(cfg.JWTAccessSecret),
		refreshSecret: []byte(cfg.JWTRefreshSecret),
		accessTTL:     cfg.AccessTokenTTL,
		refreshTTL:    cfg.RefreshTokenTTL,
	}
}

func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, role string) (SignedToken, error) {
	return m.generate(userID, role, "access", uuid.NewString(), m.accessTTL, m.accessSecret)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, role string, tokenID string) (SignedToken, error) {
	return m.generate(userID, role, "refresh", tokenID, m.refreshTTL, m.refreshSecret)
}

func (m *JWTManager) ParseAccessToken(token string) (*Claims, error) {
	return m.parse(token, "access", m.accessSecret)
}

func (m *JWTManager) ParseRefreshToken(token string) (*Claims, error) {
	return m.parse(token, "refresh", m.refreshSecret)
}

func (m *JWTManager) generate(userID uuid.UUID, role, tokenType, tokenID string, ttl time.Duration, secret []byte) (SignedToken, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	claims := Claims{
		UserID:    userID.String(),
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return SignedToken{}, err
	}

	return SignedToken{
		Token:     token,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}, nil
}

func (m *JWTManager) parse(tokenValue, expectedType string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	}, jwt.WithIssuer(m.issuer))

	if err != nil || token == nil || !token.Valid || claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
