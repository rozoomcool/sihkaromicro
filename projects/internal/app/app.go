package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/projects/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/projects/internal/config"
	"github.com/rozoomcool/sihkaromicro/projects/internal/repository"
	"github.com/rozoomcool/sihkaromicro/projects/internal/service"
	"gorm.io/gorm"
)

type App struct {
	GRPCServer *grpcapp.App
	DB         *gorm.DB
}

func NewApp(
	log *slog.Logger,
	db *gorm.DB,
	cfg *config.Config,
) *App {

	// Repository Initialization
	userRepo := repository.NewProjectRepository(db)

	projectService := service.NewProjectService(userRepo)
	grpcServer := grpcapp.New(log, projectService, cfg)

	return &App{GRPCServer: grpcServer, DB: db}
}
