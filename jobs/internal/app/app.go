package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/jobs/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/jobs/internal/config"
)

type App struct {
	GRPCServer *grpcapp.App
	log        *slog.Logger
}

func NewApp(
	cfg *config.Config,
	log *slog.Logger,
) *App {
	grpcServer := grpcapp.New(log, cfg)

	return &App{GRPCServer: grpcServer, log: log}
}
