package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/auth/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
)

type App struct {
	GRPCServer *grpcapp.App
	cfg        *config.Config
}

func NewApp(
	log *slog.Logger,
	cfg *config.Config,
) *App {
	grpcServer := grpcapp.New(log, cfg)

	return &App{GRPCServer: grpcServer, cfg: cfg}
}
