package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	pb "github.com/rozoomcool/sihkaromicro/proto/sources"
	"github.com/rozoomcool/sihkaromicro/sources/internal/apperr"
	"github.com/rozoomcool/sihkaromicro/sources/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	"github.com/rozoomcool/sihkaromicro/sources/internal/service"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const maxSourcesPerProject = 20
const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB

var allowedContentTypes = map[string]bool{
	"application/pdf": true,
	"text/plain":      true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/markdown": true,
}

type SourceHandler struct {
	pb.UnimplementedSourcesServiceServer
	srv            service.SourceService
	repo           repository.SourceRepository
	projectsClient service.ProjectsClient
	minio          service.StorageService
	producer       service.MessageProducer
	log            *slog.Logger
}

func NewSourceHandler(
	srv service.SourceService,
	repo repository.SourceRepository,
	projectsClient service.ProjectsClient,
	minio service.StorageService,
	producer service.MessageProducer,
	log *slog.Logger,
) *SourceHandler {
	return &SourceHandler{
		srv:            srv,
		repo:           repo,
		projectsClient: projectsClient,
		minio:          minio,
		producer:       producer,
		log:            log,
	}
}

func (h *SourceHandler) Register(server *grpc.Server) {
	pb.RegisterSourcesServiceServer(server, h)
}

// handler.go

func (h *SourceHandler) UploadSource(stream pb.SourcesService_UploadSourceServer) error {
	ctx := stream.Context()
	op := "SourceHandler.UploadSource"
	userID := interceptor.MustUserIDFromCtx(ctx)

	log := h.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	// Extract metadata
	first, err := stream.Recv()
	if err != nil {
		log.Error("failed to receive metadata", sl.Err(err))
		return status.Error(codes.InvalidArgument, "failed to receive metadata")
	}

	meta := first.GetMeta()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must contain metadata")
	}

	buf, contentType, err := receiveChunks(stream)
	if err != nil {
		log.Error("Error while receiveChunks", sl.Err(err))
		return err
	}

	result, err := h.srv.UploadSource(ctx, service.UploadSourceRequest{
		ProjectID:   meta.ProjectId,
		OwnerID:     userID,
		Name:        meta.Name,
		Type:        protoTypeToModel(meta.Type),
		Size:        meta.Size,
		ContentType: contentType,
		Reader:      buf,
	})
	if err != nil {
		return apperr.ToGRPC(err)
	}

	return stream.SendAndClose(&pb.UploadSourceResponse{
		SourceId: result.SourceID,
		JobId:    result.JobID,
	})
}

func (h *SourceHandler) RetryJob(ctx context.Context, req *pb.RetryJobRequest) (*pb.RetryJobResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	source, err := h.repo.FindByProjectIDAndOwnerID(ctx, req.SourceId, req.ProjectId, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "source not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Retry только если UPLOADED или FAILED
	if source.Status != model.StatusUploaded && source.Status != model.StatusFailed {
		return nil, status.Errorf(codes.FailedPrecondition, "source is not in retryable state: %s", source.Status)
	}

	jobID := uuid.New().String()
	if err := h.publishCreateJob(ctx, jobID, userID, source.ID, source.MinioPath, string(source.Type)); err != nil {
		h.log.Error("failed to publish retry job", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "failed to publish job")
	}

	if err := h.repo.UpdateStatusByJobID(ctx, source.ID, model.StatusPending, jobID); err != nil {
		h.log.Error("failed to update status", slog.Any("error", err))
	}

	return &pb.RetryJobResponse{JobId: jobID}, nil
}

func (h *SourceHandler) GetSource(ctx context.Context, req *pb.GetSourceRequest) (*pb.SourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	source, err := h.repo.FindByProjectIDAndOwnerID(ctx, req.Id, req.ProjectId, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "source not found")
	}
	if err != nil {
		h.log.Error("failed to get source", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	proto := source.ToProto()
	if source.MinioPath != "" {
		url, err := h.minio.PresignedURL(ctx, source.MinioPath)
		if err != nil {
			h.log.Error("failed to generate presigned url", slog.Any("error", err))
		} else {
			proto.Url = url
		}
	}

	return &pb.SourceResponse{Source: proto}, nil
}

func (h *SourceHandler) ListSources(ctx context.Context, req *pb.ListSourcesRequest) (*pb.ListSourcesResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	sources, err := h.repo.FindAllByProjectIDAndOwnerID(ctx, req.ProjectId, userID)
	if err != nil {
		h.log.Error("failed to list sources", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	result := make([]*pb.Source, len(sources))
	for i, s := range sources {
		proto := s.ToProto()
		if s.MinioPath != "" {
			url, err := h.minio.PresignedURL(ctx, s.MinioPath)
			if err != nil {
				h.log.Error("failed to generate presigned url", slog.Any("error", err))
			} else {
				proto.Url = url
			}
		}
		result[i] = proto
	}

	return &pb.ListSourcesResponse{Sources: result}, nil
}

func (h *SourceHandler) DeleteSource(ctx context.Context, req *pb.DeleteSourceRequest) (*pb.DeleteSourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	source, err := h.repo.FindByProjectIDAndOwnerID(ctx, req.Id, req.ProjectId, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "source not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Удаляем source из БД
	if err := h.repo.DeleteByProjectIDAndOwnerID(ctx, req.Id, req.ProjectId, userID); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Отменяем job если есть активный
	if source.JobID != "" && (source.Status == model.StatusPending || source.Status == model.StatusProcessing) {
		if err := h.publishCancelJob(ctx, source.JobID, userID, source.ID); err != nil {
			h.log.Error("failed to publish cancel job", slog.Any("error", err))
		}
	}

	// Удаляем из MinIO
	if source.MinioPath != "" {
		if err := h.minio.Delete(ctx, source.MinioPath); err != nil {
			h.log.Error("failed to delete from minio", slog.Any("error", err))
		}
	}

	return &pb.DeleteSourceResponse{Success: true}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (h *SourceHandler) publishCreateJob(ctx context.Context, jobID, ownerID string, sourceID int64, minioPath, fileType string) error {
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

	return h.producer.Publish(ctx, "jobs.create", jobID, string(payload))
}

func (h *SourceHandler) publishCancelJob(ctx context.Context, jobID, ownerID string, sourceID int64) error {
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

	return h.producer.Publish(ctx, "jobs.cancel", jobID, string(payload))
}

func protoTypeToModel(t pb.SourceType) model.SourceType {
	switch t {
	case pb.SourceType_SOURCE_TYPE_PDF:
		return model.TypePDF
	case pb.SourceType_SOURCE_TYPE_TXT:
		return model.TypeTXT
	case pb.SourceType_SOURCE_TYPE_DOCX:
		return model.TypeDOCX
	case pb.SourceType_SOURCE_TYPE_MARKDOWN:
		return model.TypeMarkdown
	case pb.SourceType_SOURCE_TYPE_URL:
		return model.TypeURL
	default:
		return model.TypeTXT
	}
}

func sourceTypeToContentType(t model.SourceType) string {
	switch t {
	case model.TypePDF:
		return "application/pdf"
	case model.TypeDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case model.TypeMarkdown:
		return "text/markdown"
	default:
		return "text/plain"
	}
}

func receiveChunks(stream pb.SourcesService_UploadSourceServer) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	var totalSize int64
	var contentType string
	firstChunk := true
	doneReceived := false

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", status.Error(codes.Internal, "stream error")
		}

		if msg.GetDone() {
			doneReceived = true
			break
		}

		chunk := msg.GetChunk()
		if chunk == nil {
			return nil, "", status.Error(codes.InvalidArgument, "expected chunk or done signal")
		}

		if firstChunk {
			if len(chunk) < 512 {
				return nil, "", status.Error(codes.InvalidArgument, "first chunk too small for validation")
			}
			contentType = http.DetectContentType(chunk)
			if !allowedContentTypes[contentType] {
				return nil, "", status.Errorf(codes.InvalidArgument, "invalid file type: %s", contentType)
			}
			firstChunk = false
		}

		totalSize += int64(len(chunk))
		if totalSize > maxFileSize {
			return nil, "", status.Error(codes.InvalidArgument, "file too large")
		}

		buf.Write(chunk)
	}

	if !doneReceived {
		return nil, "", status.Error(codes.Canceled, "upload incomplete: connection lost")
	}
	if buf.Len() == 0 {
		return nil, "", status.Error(codes.InvalidArgument, "empty file")
	}

	return buf, contentType, nil
}
