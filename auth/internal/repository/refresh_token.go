package repository

import (
	"context"
	"errors"
	"time"

	"github.com/rozoomcool/sihkaromicro/auth/internal/model"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteExpiredTokensForUser(ctx context.Context, userID int64) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// GetRefreshToken implements RefreshTokenRepository.
func (r *refreshTokenRepository) GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("Token not found")
		}
		return nil, err
	}
	return &refreshToken, nil
}

// SaveRefreshToken implements RefreshTokenRepository.
func (r *refreshTokenRepository) SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	refreshToken := model.RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&refreshToken).Error
}

// DeleteExpiredTokens implements RefreshTokenRepository.
func (r *refreshTokenRepository) DeleteExpiredTokensForUser(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND expires_at < ?", userID, time.Now()).
		Delete(&model.RefreshToken{}).Error
}

// DeleteRefreshToken implements RefreshTokenRepository.
func (r *refreshTokenRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&model.RefreshToken{}).Error
}
