package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/sources/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/internal/infrastructure/broker"
	"github.com/rozoomcool/sihkaromicro/sources/internal/infrastructure/clients"
	"github.com/rozoomcool/sihkaromicro/sources/internal/infrastructure/storage"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	"github.com/rozoomcool/sihkaromicro/sources/internal/service"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
	"gorm.io/gorm"
)

type App struct {
	GRPCServer *grpcapp.App
	producer   service.MessageProducer
	log        *slog.Logger
}

func NewApp(cfg *config.Config, log *slog.Logger, db *gorm.DB) *App {
	sourceRepo := repository.NewSourceRepository(db)

	minioClient, err := storage.NewMinioClient(cfg.MinIO)
	if err != nil {
		panic("unable to connect to MinIO: " + err.Error())
	}

	kafkaProducer := broker.NewKafkaProducer(cfg.Kafka)

	projectsClient, err := clients.NewProjectsClient(cfg.ProjectsUrl)
	if err != nil {
		panic("unable to connect to projects service: " + err.Error())
	}

	sourceService := service.NewSourceService(
		sourceRepo,
		projectsClient,
		kafkaProducer,
		log,
		minioClient,
	)

	grpcServer := grpcapp.New(sourceService, log, cfg)

	return &App{
		GRPCServer: grpcServer,
		producer:   kafkaProducer,
		log:        log,
	}
}

func (a *App) Stop() {
	a.GRPCServer.Stop()
	if err := a.producer.Close(); err != nil {
		a.log.Error("failed to close kafka producer", sl.Err(err))
	}
}
