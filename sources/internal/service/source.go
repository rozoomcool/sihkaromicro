package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/rozoomcool/sihkaromicro/sources/internal/apperr"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
)

type UploadSourceRequest struct {
	ProjectID   int64
	OwnerID     string
	Name        string
	Type        model.SourceType
	Size        int64
	ContentType string
	Reader      io.Reader
}

type UploadSourceResult struct {
	SourceID int64
	JobID    string
}

const maxSourcesPerProject = 20
const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB

type SourceService interface {
	CheckProjectAccess(ctx context.Context, projectID int64, ownerID string) error
	AddSource(ctx context.Context, source *model.Source) error
	CheckLimits(ctx context.Context, projectID int64, userID string) error
	UploadSource(ctx context.Context, req UploadSourceRequest) (*UploadSourceResult, error)
}

type sourceService struct {
	sourceRepo     repository.SourceRepository
	projectsClient ProjectsClient
	producer       MessageProducer
	log            *slog.Logger
	storage        StorageService
}

func NewSourceService(
	sourceRepo repository.SourceRepository,
	projectsClient ProjectsClient,
	prducer MessageProducer,
	log *slog.Logger,
	minioClient StorageService,
) SourceService {
	return &sourceService{
		sourceRepo:     sourceRepo,
		projectsClient: projectsClient,
		log:            log,
		storage:        minioClient,
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

func (s *sourceService) UploadSource(ctx context.Context, req UploadSourceRequest) (*UploadSourceResult, error) {
	op := "SourceService.UploadSource"

	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", req.OwnerID),
		slog.String("projectID", fmt.Sprintf("%d", req.ProjectID)),
	)

	// Check limits
	if err := s.CheckProjectAccess(ctx, req.ProjectID, req.OwnerID); err != nil {
		return nil, err
	}
	if err := s.CheckLimits(ctx, req.ProjectID, req.OwnerID); err != nil {
		return nil, err
	}

	// Save source with UPLOADING status
	source := &model.Source{
		ProjectID: req.ProjectID,
		OwnerID:   req.OwnerID,
		Name:      req.Name,
		Type:      req.Type,
		Status:    model.StatusUploading,
		Size:      req.Size,
	}
	if err := s.sourceRepo.Save(ctx, source); err != nil {
		log.Error("failed to save source", sl.Err(err))
		return nil, apperr.Unavailable("source repository")
	}

	// Delete record if error
	var success bool
	defer func() {
		if !success {
			if err := s.sourceRepo.DeleteByProjectIDAndOwnerID(
				context.Background(), source.ID, source.ProjectID, source.OwnerID,
			); err != nil {
				log.Error("failed to compensate: delete source", sl.Err(err))
			}
		}
	}()

	// Upload file in storage
	objectName := s.storage.ObjectName(req.OwnerID, source.ID, req.Name)
	if err := s.storage.Upload(ctx, objectName, req.Reader, req.Size, req.ContentType); err != nil {
		log.Error("failed to upload to storage", sl.Err(err))
		return nil, apperr.Unavailable("storage")
	}

	// Delete file from storage if !success
	defer func() {
		if !success {
			if err := s.storage.Delete(context.Background(), objectName); err != nil {
				log.Error("failed to compensate: delete from storage", sl.Err(err))
			}
		}
	}()

	// Update path and status to UPLOADED
	if err := s.sourceRepo.UpdateMinioPath(ctx, source.ID, objectName); err != nil {
		log.Error("failed to update minio path", sl.Err(err))
		return nil, apperr.Unavailable("source repository")
	}
	if err := s.sourceRepo.UpdateStatusByJobID(ctx, source.ID, model.StatusUploaded, ""); err != nil {
		log.Error("failed to update status to uploaded", sl.Err(err))
		return nil, apperr.Unavailable("source repository")
	}

	// Publish job in producer
	jobID := uuid.New().String()
	if err := s.publishCreateJob(ctx, jobID, req.OwnerID, source.ID, objectName, string(req.Type)); err != nil {
		log.Error("failed to publish job, client can retry", sl.Err(err))
		success = true
		return &UploadSourceResult{
			SourceID: source.ID,
			JobID:    "",
		}, nil
	}

	// Update status to PENDING
	if err := s.sourceRepo.UpdateStatusByJobID(ctx, source.ID, model.StatusPending, jobID); err != nil {
		log.Error("failed to update status to pending", sl.Err(err))
	}

	success = true

	return &UploadSourceResult{
		SourceID: source.ID,
		JobID:    jobID,
	}, nil
}

func (s *sourceService) publishCreateJob(ctx context.Context, jobID, ownerID string, sourceID int64, minioPath, fileType string) error {
	type CreateJobEvent struct {
		JobID     string `json:"job_id"`
		Type      string `json:"type"`
		OwnerID   string `json:"owner_id"`
		SourceID  int64  `json:"source_id"`
		MinioPath string `json:"minio_path"`
		FileType  string `json:"file_type"`
	}

	payload, err := json.Marshal(CreateJobEvent{
		JobID:     jobID,
		Type:      "chunking",
		OwnerID:   ownerID,
		SourceID:  sourceID,
		MinioPath: minioPath,
		FileType:  fileType,
	})
	if err != nil {
		return err
	}

	return s.producer.Publish(ctx, "jobs.create", jobID, string(payload))
}
