package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rozoomcool/sihkaromicro/sources/internal/app"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/database"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env, cfg.LogFile)

	log.Info("starting application", slog.String("env", cfg.Env))

	db, err := database.New(cfg.DB)
	if err != nil {
		log.Error("failed to initialize database", sl.Err(err))
		os.Exit(1)
	}

	application := app.NewApp(cfg, log, db)

	go func() { application.GRPCServer.MustRun() }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	application.Stop()
	log.Info("gracefully stopped")
}
