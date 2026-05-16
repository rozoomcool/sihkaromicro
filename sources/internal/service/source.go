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

const MaxSourcesPerProject = 20
const MaxFileSize = 1 * 1024 * 1024 * 1024 // 1GB

type UploadSourceRequest struct {
	ProjectID   int64
	OwnerID     string
	Name        string
	Type        model.SourceType
	Size        int64
	ContentType string
	Reader      io.Reader // nil for TypeURL
	SourceURL   string    // only for TypeURL
}

type UploadSourceResult struct {
	SourceID int64
	JobID    string
}

type SourceDetail struct {
	Source    *model.Source
	SignedURL string
}

type SourceService interface {
	UploadSource(ctx context.Context, req UploadSourceRequest) (*UploadSourceResult, error)
	GetSource(ctx context.Context, id, projectID int64, ownerID string) (*SourceDetail, error)
	ListSources(ctx context.Context, projectID int64, ownerID string) ([]*SourceDetail, error)
	DeleteSource(ctx context.Context, id, projectID int64, ownerID string) error
	RetryJob(ctx context.Context, id, projectID int64, ownerID string) (string, error)
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
	producer MessageProducer,
	log *slog.Logger,
	storage StorageService,
) SourceService {
	return &sourceService{
		sourceRepo:     sourceRepo,
		projectsClient: projectsClient,
		producer:       producer,
		log:            log,
		storage:        storage,
	}
}

func (s *sourceService) GetSource(ctx context.Context, id, projectID int64, ownerID string) (*SourceDetail, error) {
	source, err := s.sourceRepo.FindByProjectIDAndOwnerID(ctx, id, projectID, ownerID)
	if err != nil {
		return nil, err
	}

	detail := &SourceDetail{Source: source}
	if source.MinioPath != "" {
		url, err := s.storage.PresignedURL(ctx, source.MinioPath)
		if err != nil {
			s.log.Error("failed to generate presigned url", sl.Err(err))
		} else {
			detail.SignedURL = url
		}
	}
	return detail, nil
}

func (s *sourceService) ListSources(ctx context.Context, projectID int64, ownerID string) ([]*SourceDetail, error) {
	sources, err := s.sourceRepo.FindAllByProjectIDAndOwnerID(ctx, projectID, ownerID)
	if err != nil {
		return nil, err
	}

	result := make([]*SourceDetail, len(sources))
	for i := range sources {
		detail := &SourceDetail{Source: &sources[i]}
		if sources[i].MinioPath != "" {
			url, err := s.storage.PresignedURL(ctx, sources[i].MinioPath)
			if err != nil {
				s.log.Error("failed to generate presigned url", sl.Err(err))
			} else {
				detail.SignedURL = url
			}
		}
		result[i] = detail
	}
	return result, nil
}

func (s *sourceService) DeleteSource(ctx context.Context, id, projectID int64, ownerID string) error {
	source, err := s.sourceRepo.FindByProjectIDAndOwnerID(ctx, id, projectID, ownerID)
	if err != nil {
		return err
	}

	if err := s.sourceRepo.DeleteByProjectIDAndOwnerID(ctx, id, projectID, ownerID); err != nil {
		return err
	}

	// Best-effort: cancel job if still active
	if source.JobID != "" && (source.Status == model.StatusPending || source.Status == model.StatusProcessing) {
		if err := s.publishCancelJob(ctx, source.JobID, ownerID, source.ID); err != nil {
			s.log.Error("failed to publish cancel job", sl.Err(err))
		}
	}

	// Best-effort: delete file from storage
	if source.MinioPath != "" {
		if err := s.storage.Delete(context.Background(), source.MinioPath); err != nil {
			s.log.Error("failed to delete from storage", sl.Err(err))
		}
	}

	return nil
}

func (s *sourceService) RetryJob(ctx context.Context, id, projectID int64, ownerID string) (string, error) {
	source, err := s.sourceRepo.FindByProjectIDAndOwnerID(ctx, id, projectID, ownerID)
	if err != nil {
		return "", err
	}

	if source.Status != model.StatusUploaded && source.Status != model.StatusFailed {
		return "", apperr.InvalidArgument(fmt.Sprintf("source is not in retryable state: %s", source.Status))
	}

	jobID := uuid.New().String()
	if err := s.publishCreateJob(ctx, jobID, ownerID, source.ID, source.MinioPath, string(source.Type)); err != nil {
		s.log.Error("failed to publish retry job", sl.Err(err))
		return "", apperr.Unavailable("message broker")
	}

	if err := s.sourceRepo.UpdateStatusBySourceID(ctx, source.ID, model.StatusPending, jobID); err != nil {
		s.log.Error("failed to update status after retry", sl.Err(err))
	}

	return jobID, nil
}

func (s *sourceService) UploadSource(ctx context.Context, req UploadSourceRequest) (*UploadSourceResult, error) {
	op := "SourceService.UploadSource"
	log := s.log.With(
		slog.String("op", op),
		slog.String("userID", req.OwnerID),
		slog.Int64("projectID", req.ProjectID),
	)

	if err := s.checkProjectAccess(ctx, req.ProjectID, req.OwnerID); err != nil {
		return nil, err
	}
	if err := s.checkLimits(ctx, req.ProjectID, req.OwnerID); err != nil {
		return nil, err
	}

	source := &model.Source{
		ProjectID: req.ProjectID,
		OwnerID:   req.OwnerID,
		Name:      req.Name,
		Type:      req.Type,
		Status:    model.StatusUploading,
		Size:      req.Size,
		SourceURL: req.SourceURL,
	}
	if err := s.sourceRepo.Save(ctx, source); err != nil {
		log.Error("failed to save source", sl.Err(err))
		return nil, apperr.Unavailable("source repository")
	}

	var success bool
	defer func() {
		if !success {
			if err := s.sourceRepo.DeleteByProjectIDAndOwnerID(
				context.Background(), source.ID, source.ProjectID, source.OwnerID,
			); err != nil {
				log.Error("failed to compensate: delete source record", sl.Err(err))
			}
		}
	}()

	// URL type: skip file upload, go straight to job publishing
	if req.Type != model.TypeURL {
		objectName := s.storage.ObjectName(req.OwnerID, source.ID, req.Name)
		if err := s.storage.Upload(ctx, objectName, req.Reader, req.Size, req.ContentType); err != nil {
			log.Error("failed to upload to storage", sl.Err(err))
			return nil, apperr.Unavailable("storage")
		}

		defer func() {
			if !success {
				if err := s.storage.Delete(context.Background(), objectName); err != nil {
					log.Error("failed to compensate: delete from storage", sl.Err(err))
				}
			}
		}()

		if err := s.sourceRepo.UpdateMinioPath(ctx, source.ID, objectName); err != nil {
			log.Error("failed to update minio path", sl.Err(err))
			return nil, apperr.Unavailable("source repository")
		}
		source.MinioPath = objectName
	}

	if err := s.sourceRepo.UpdateStatusBySourceID(ctx, source.ID, model.StatusUploaded, ""); err != nil {
		log.Error("failed to update status to uploaded", sl.Err(err))
		return nil, apperr.Unavailable("source repository")
	}

	jobID := uuid.New().String()
	if err := s.publishCreateJob(ctx, jobID, req.OwnerID, source.ID, source.MinioPath, string(req.Type)); err != nil {
		// Upload succeeded — client can retry the job separately
		log.Error("failed to publish job, client can retry via RetryJob", sl.Err(err))
		success = true
		return &UploadSourceResult{SourceID: source.ID, JobID: ""}, nil
	}

	if err := s.sourceRepo.UpdateStatusBySourceID(ctx, source.ID, model.StatusPending, jobID); err != nil {
		log.Error("failed to update status to pending", sl.Err(err))
	}

	success = true
	return &UploadSourceResult{SourceID: source.ID, JobID: jobID}, nil
}

func (s *sourceService) checkProjectAccess(ctx context.Context, projectID int64, ownerID string) error {
	hasAccess, err := s.projectsClient.CheckAccess(ctx, projectID)
	if err != nil {
		s.log.Error("projects client error", sl.Err(err), slog.String("userID", ownerID))
		return apperr.Unavailable("projects service")
	}
	if !hasAccess {
		return apperr.ErrPermissionDenied
	}
	return nil
}

func (s *sourceService) checkLimits(ctx context.Context, projectID int64, userID string) error {
	count, err := s.sourceRepo.CountByProjectIDAndOwnerID(ctx, projectID, userID)
	if err != nil {
		return apperr.Unavailable("source repository")
	}
	if count >= MaxSourcesPerProject {
		return apperr.LimitExceeded("max sources per project reached")
	}
	return nil
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

func (s *sourceService) publishCancelJob(ctx context.Context, jobID, ownerID string, sourceID int64) error {
	type CancelJobEvent struct {
		JobID    string `json:"job_id"`
		OwnerID  string `json:"owner_id"`
		SourceID int64  `json:"source_id"`
	}
	payload, err := json.Marshal(CancelJobEvent{
		JobID:    jobID,
		OwnerID:  ownerID,
		SourceID: sourceID,
	})
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, "jobs.cancel", jobID, string(payload))
}
