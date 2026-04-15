package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/rozoomcool/sihkaromicro/auth/internal/app"
	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/database"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/logger"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/logger/sl"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env, cfg.LogFile)

	log.Info("Start running application", "Mode", cfg.Env, "Logs", cfg.LogFile)

	log.Info("Configs loaded")

	db, err := database.New(cfg.DB)
	if err != nil {
		log.Error("Error when loading database", sl.Err(err))
		panic(err)
	}
	if err = database.AutoMigrate(db); err != nil {
		log.Error("Failed auto migrate", sl.Err(err))
	}
	log.Info("Database successfully initialized")

	port, err := strconv.Atoi(cfg.GRPC.Port)
	if err != nil {
		panic(err)
	}

	app := app.NewApp(log, db, port, cfg.JWT)

	go func() { app.GRPCServer.MustRun() }()

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	app.GRPCServer.Stop()
	log.Info("Gracefully stopped")

}
