package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rozoomcool/sihkaromicro/auth/internal/model"
	"github.com/rozoomcool/sihkaromicro/auth/internal/repository"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(ctx context.Context, username string, password string) (ID int64, err error)
	Login(ctx context.Context, username string, password string) (*AuthResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
	rtRepo   repository.RefreshTokenRepository
	jwtMgr   jwt.TokenManager
	hasher   jwt.PasswordHasher
	jwtTTL   time.Duration
	rtTTL    time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtMgr jwt.TokenManager,
	hasher jwt.PasswordHasher,
	jwtTTL, rtTTL time.Duration,
) AuthService {
	return &authService{
		userRepo: userRepo,
		rtRepo:   refreshTokenRepo,
		jwtMgr:   jwtMgr,
		hasher:   hasher,
		jwtTTL:   jwtTTL,
		rtTTL:    rtTTL,
	}
}

type RegisterInput struct {
	Username string
	Password string
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	UserID       int64
}

func (s *authService) Register(ctx context.Context, username, password string) (int64, error) {
	// Password hashing
	hashedPwd, err := s.hasher.Hash(password)
	if err != nil {
		return 0, status.Error(codes.Internal, "failed to hash password")
	}

	user := &model.User{
		Username: username,
		Password: hashedPwd,
	}
	// Creating user
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return 0, status.Error(codes.AlreadyExists, "username already registered")
		}
		return 0, status.Error(codes.Internal, "failed to register user")
	}

	return user.ID, nil
}

func (s *authService) Login(ctx context.Context, username, password string) (*AuthResponse, error) {
	// Check user existing
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	// Verify password
	if err := s.hasher.Verify(password, user.Password); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Creating new refresh token
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(s.rtTTL)

	if err := s.rtRepo.SaveRefreshToken(ctx, user.ID, refreshToken, expiresAt); err != nil {
		return nil, status.Error(codes.Internal, "failed to save session")
	}
	// Generate new jwt token
	accessToken, err := s.jwtMgr.NewJWT(user.ID, user.Username, s.jwtTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       user.ID,
	}, nil
}

func (s *authService) Refresh(ctx context.Context, oldRefreshToken string) (*AuthResponse, error) {
	// Проверяем токен в БД
	rt, err := s.rtRepo.GetRefreshToken(ctx, oldRefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	// Получаем пользователя
	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	// Удаляем старый токен (rotation)
	if err := s.rtRepo.DeleteRefreshToken(ctx, oldRefreshToken); err != nil {
		// Логируем, но не прерываем
	}

	// Создаем новую пару
	newRefreshToken := uuid.New().String()
	newExpiresAt := time.Now().Add(s.rtTTL)

	if err := s.rtRepo.SaveRefreshToken(ctx, user.ID, newRefreshToken, newExpiresAt); err != nil {
		return nil, status.Error(codes.Internal, "failed to save new session")
	}

	newAccessToken, err := s.jwtMgr.NewJWT(user.ID, user.Username, s.jwtTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		UserID:       user.ID,
	}, nil
}

func (s *authService) Validate(ctx context.Context, accessToken string) (bool, int64, string, error) {
	userID, email, err := s.jwtMgr.Verify(accessToken)
	if err != nil {
		return false, 0, "", nil // Не ошибка, просто невалидный токен
	}

	// Опционально: проверить, не заблокирован ли пользователь
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, 0, "", nil
	}

	return true, userID, email, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.rtRepo.DeleteRefreshToken(ctx, refreshToken)
}
