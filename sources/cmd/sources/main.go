package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rozoomcool/sihkaromicro/sources/internal/app"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env, cfg.LogFile)

	log.Info("Start running application", "Mode", cfg.Env, "Logs", cfg.LogFile)

	log.Info("Configs loaded")

	app := app.NewApp(cfg, log)

	go func() { app.GRPCServer.MustRun() }()

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	app.GRPCServer.Stop()
	log.Info("Gracefully stopped")
}
