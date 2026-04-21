package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rozoomcool/sihkaromicro/auth/internal/app"
	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env, cfg.LogFile)

	log.Info("Start running application", "Mode", cfg.Env, "Logs", cfg.LogFile)

	log.Info("Configs loaded")

	log.Info(cfg.Keycloak.Client.ID)
	log.Info(cfg.Keycloak.Url)
	log.Info(cfg.Keycloak.Realm)

	app := app.NewApp(log, cfg)

	go func() { app.GRPCServer.MustRun() }()

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	app.GRPCServer.Stop()
	log.Info("Gracefully stopped")

}
