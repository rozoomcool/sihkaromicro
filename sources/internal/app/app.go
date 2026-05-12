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
	"gorm.io/gorm"
)

type App struct {
	GRPCServer *grpcapp.App
	log        *slog.Logger
}

func NewApp(
	cfg *config.Config,
	log *slog.Logger,
	db *gorm.DB,
) *App {
	sourceRepo := repository.NewSourceRepository(db)
	minioClient, err := storage.NewMinioClient(cfg.MinIO)
	if err != nil {
		panic("Unable connect to MinIO")
	}

	kafkaProducer := broker.NewKafkaProducer(cfg.Kafka)
	projectsClient, err := clients.NewProjectsClient(cfg.ProjectsUrl)
	if err != nil {
		panic("Unable connect to Projects service")
	}
	sourceService := service.NewSourceService(
		sourceRepo,
		projectsClient,
		kafkaProducer,
		log,
		minioClient,
	)

	grpcServer := grpcapp.New(
		sourceService,
		sourceRepo,
		minioClient,
		kafkaProducer,
		projectsClient,
		log,
		cfg,
	)

	return &App{GRPCServer: grpcServer, log: log}
}
