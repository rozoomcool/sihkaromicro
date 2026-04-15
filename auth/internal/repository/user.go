package repository

import (
	"context"
	"errors"
	"time"

	"github.com/rozoomcool/sihkaromicro/auth/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	CreateWithToken(ctx context.Context, user *model.User, token string, expiresAt time.Time) error
	// DeleteByID(ctx context.Context, id int64) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Transaction of creating user and token
func (r *userRepository) CreateWithToken(ctx context.Context, user *model.User, token string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		rt := model.RefreshToken{
			UserID:    user.ID,
			Token:     token,
			ExpiresAt: expiresAt,
		}
		if err := tx.Create(&rt).Error; err != nil {
			return err
		}
		return nil
	})
}
