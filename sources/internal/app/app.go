package app

import (
	"log/slog"

	grpcapp "github.com/rozoomcool/sihkaromicro/sources/internal/app/grpc"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/internal/kafka"
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
	minioClient, err := service.NewMinioClient(cfg.MinIO)
	kafkaProducer := kafka.NewProducer(cfg.Kafka)
	projectsClient, err := service.NewProjectsClient(cfg.ProjectsUrl)
	sourceService := service.NewSourceService(sourceRepo, projectsClient, log, minioClient)
	if err != nil {
		panic(err)
	}

	if err != nil {
		panic("Unable connect to MinIO")
	}

	grpcServer := grpcapp.New(sourceService, sourceRepo, minioClient, kafkaProducer, projectsClient, log, cfg)

	return &App{GRPCServer: grpcServer, log: log}
}
