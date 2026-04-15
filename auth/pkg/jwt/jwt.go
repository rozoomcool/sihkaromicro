package jwt

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// TokenManager управляет JWT токенами
type TokenManager interface {
	NewJWT(userID int64, username string, ttl time.Duration) (string, error)
	Verify(token string) (int64, string, error)
}

// JWTClaims кастомные claims для токена
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secret),
	}
}

func (m *JWTManager) NewJWT(userID int64, username string, ttl time.Duration) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
			Subject:   strconv.FormatInt(userID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) Verify(tokenString string) (int64, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, "", ErrExpiredToken
		}
		return 0, "", ErrInvalidToken
	}

	if !token.Valid {
		return 0, "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return 0, "", ErrInvalidToken
	}

	return claims.UserID, claims.Username, nil
}
