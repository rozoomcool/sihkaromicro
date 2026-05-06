package service

import (
	"context"
	"log/slog"

	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxSourcesPerProject = 20
const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB

type SourceService interface {
	AddSource(ctx context.Context, projectID int64, ownerID string) (*model.Source, error)
}

type sourceService struct {
	sourceRepo  repository.SourceRepository
	log         *slog.Logger
	minioClient *MinioClient
}

func NewSourceService(
	sourceRepo repository.SourceRepository,
	log *slog.Logger,
	minioClient *MinioClient,
) SourceService {
	return &sourceService{
		sourceRepo:  sourceRepo,
		log:         log,
		minioClient: minioClient,
	}
}

// AddSource implements SourceService.
func (s *sourceService) AddSource(ctx context.Context, projectID int64, userID string) (*model.Source, error) {
	op := "SourceService.AddSource"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	count, err := s.sourceRepo.CountByProjectIDAndOwnerID(ctx, projectID, userID)
	if err != nil {
		log.Error("Check limits", sl.Err(err))
		return status.Error(codes.Internal, "internal error")
	}
	if count >= maxSourcesPerProject {
		log.Error("Max sources uploaded", sl.Err(err))
		return status.Errorf(codes.FailedPrecondition, "project has reached the limit of %d sources", maxSourcesPerProject)
	}
}
