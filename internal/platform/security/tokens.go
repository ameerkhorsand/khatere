package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTTokenService struct {
	secret    []byte
	accessTTL time.Duration
}

func NewJWTTokenService(secret []byte, accessTTL time.Duration) *JWTTokenService {
	return &JWTTokenService{secret: secret, accessTTL: accessTTL}
}

type accessClaims struct {
	jwt.RegisteredClaims
	AccountType domain.AccountType `json:"account_type"`
}

func (s *JWTTokenService) IssueAccessToken(accountID uuid.UUID, accountType domain.AccountType) (string, error) {
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accountID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		AccountType: accountType,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTTokenService) GenerateRefreshTokenValue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *JWTTokenService) HashRefreshToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
