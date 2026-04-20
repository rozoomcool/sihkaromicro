package logger

import (
	"log/slog"
	"os"

	slogpretty "github.com/rozoomcool/sihkaromicro/projects/pkg/logger/handler"
	slogmulti "github.com/samber/slog-multi"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func SetupLogger(env string, logFile string) *slog.Logger {
	out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	var log *slog.Logger
	var level slog.Level

	switch env {
	case envDev:
		level = slog.LevelDebug
	case envProd:
		level = slog.LevelInfo
	}

	prettyOpts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: level,
		},
	}

	prettyHandler := prettyOpts.NewPrettyHandler(os.Stdout)
	fileHandler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})

	log = slog.New(
		slogmulti.Fanout(
			prettyHandler,
			fileHandler,
		),
	)

	return log
}
