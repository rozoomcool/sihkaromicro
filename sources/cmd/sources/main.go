package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rozoomcool/sihkaromicro/sources/internal/app"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/database"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env, cfg.LogFile)

	log.Info("Start running application", "Mode", cfg.Env, "Logs", cfg.LogFile)

	log.Info("Configs loaded")
	log.Info("MINIO CFGS", slog.String("ID", cfg.MinIO.AccessKeyID), slog.String("Endpoint", cfg.MinIO.Endpoint))

	db, err := database.New(cfg.DB)
	if err != nil {
		log.Error("Error initialize database", sl.Err(err))
		panic(err)
	}

	err = db.AutoMigrate(model.Source{})
	if err != nil {
		log.Error("Failed to init migrations", sl.Err(err))
		panic(err)
	}

	app := app.NewApp(cfg, log, db)

	go func() { app.GRPCServer.MustRun() }()

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	app.GRPCServer.Stop()
	log.Info("Gracefully stopped")
}
