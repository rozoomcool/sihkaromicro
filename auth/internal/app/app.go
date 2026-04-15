package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/auth/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
	"github.com/rozoomcool/sihkaromicro/auth/internal/repository"
	"github.com/rozoomcool/sihkaromicro/auth/internal/service"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/jwt"
	"gorm.io/gorm"
)

type App struct {
	GRPCServer *grpcapp.App
	DB         *gorm.DB
}

func NewApp(
	log *slog.Logger,
	db *gorm.DB,
	grpcPort int,
	jwtCfg config.JWTConf,
) *App {

	// Repository Initialization
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// JWT services initialization
	jwtMgr := jwt.NewJWTManager(jwtCfg.Secret)
	hasher := jwt.NewBcryptHasher(10)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtMgr, hasher, jwtCfg.AccessTokenTTL, jwtCfg.RefreshTokenTTL)
	grpcServer := grpcapp.New(log, authService, grpcPort)

	return &App{GRPCServer: grpcServer, DB: db}
}
