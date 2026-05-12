package service

import (
	"context"
	"log/slog"

	"github.com/rozoomcool/sihkaromicro/sources/internal/apperr"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
)

const maxSourcesPerProject = 20
const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB

type SourceService interface {
	CheckProjectAccess(ctx context.Context, projectID int64, ownerID string) error
	AddSource(ctx context.Context, source *model.Source) error
	CheckLimits(ctx context.Context, projectID int64, userID string) error
}

type sourceService struct {
	sourceRepo     repository.SourceRepository
	projectsClient ProjectsClient
	log            *slog.Logger
	minioClient    *MinioClient
}

func NewSourceService(
	sourceRepo repository.SourceRepository,
	projectsClient ProjectsClient,
	log *slog.Logger,
	minioClient *MinioClient,
) SourceService {
	return &sourceService{
		sourceRepo:     sourceRepo,
		projectsClient: projectsClient,
		log:            log,
		minioClient:    minioClient,
	}
}

func (s *sourceService) CheckProjectAccess(ctx context.Context, projectID int64, ownerID string) error {
	op := "SourceService.CheckProjectAccess"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", ownerID),
	)

	hasAccess, err := s.projectsClient.CheckAccess(ctx, projectID)
	if err != nil {
		log.Error("projects client error", sl.Err(err))
		return apperr.Unavailable("projects service")
	}
	if !hasAccess {
		return apperr.ErrPermissionDenied
	}
	return nil
}

func (s *sourceService) AddSource(ctx context.Context, source *model.Source) error {
	op := "SourceService.AddSource"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", source.OwnerID),
	)

	if err := s.sourceRepo.Save(ctx, source); err != nil {
		log.Error("failed to save source", sl.Err(err))
		return apperr.Unavailable("source repository")
	}

	return nil
}

func (s *sourceService) CheckLimits(ctx context.Context, projectID int64, userID string) error {
	op := "SourceService.CheckLimits"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	count, err := s.sourceRepo.CountByProjectIDAndOwnerID(ctx, projectID, userID)
	if err != nil {
		log.Error("failed to count sources", sl.Err(err))
		return apperr.Unavailable("source repository")
	}
	if count >= maxSourcesPerProject {
		return apperr.LimitExceeded("max sources per project reached")
	}

	return nil
}
