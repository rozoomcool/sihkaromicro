package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	pb "github.com/rozoomcool/sihkaromicro/sources/gen/proto/sources"
	"github.com/rozoomcool/sihkaromicro/sources/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/sources/internal/kafka"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/repository"
	minioclient "github.com/rozoomcool/sihkaromicro/sources/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type SourceHandler struct {
	pb.UnimplementedSourcesServiceServer
	repo     repository.SourceRepository
	minio    *minioclient.MinioClient
	producer *kafka.Producer
	log      *slog.Logger
}

func NewSourceHandler(
	repo repository.SourceRepository,
	minio *minioclient.MinioClient,
	producer *kafka.Producer,
	log *slog.Logger,
) *SourceHandler {
	return &SourceHandler{
		repo:     repo,
		minio:    minio,
		producer: producer,
		log:      log,
	}
}

func (h *SourceHandler) Register(server *grpc.Server) {
	pb.RegisterSourcesServiceServer(server, h)
}

const maxSourcesPerProject = 20

func (h *SourceHandler) UploadSource(stream pb.SourcesService_UploadSourceServer) error {
	userID := interceptor.MustUserIDFromCtx(stream.Context())

	first, err := stream.Recv()
	if err != nil {
		return status.Error(codes.InvalidArgument, "failed to receive metadata")
	}

	meta := first.GetMeta()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must contain metadata")
	}

	// Проверяем лимит
	count, err := h.repo.Count(stream.Context(), meta.ProjectId, userID)
	if err != nil {
		return status.Error(codes.Internal, "internal error")
	}
	if count >= maxSourcesPerProject {
		return status.Errorf(codes.FailedPrecondition, "project has reached the limit of %d sources", maxSourcesPerProject)
	}

	// 2. Создаём запись в БД
	source := &model.Source{
		ProjectID: meta.ProjectId,
		OwnerID:   userID,
		Name:      meta.Name,
		Type:      protoTypeToModel(meta.Type),
		Status:    model.StatusPending,
		Size:      meta.Size,
		MinioPath: "",
	}

	if err := h.repo.Save(stream.Context(), source); err != nil {
		h.log.Error("failed to save source", slog.Any("error", err))
		return status.Error(codes.Internal, "failed to create source")
	}

	// 3. Стримим чанки в MinIO
	objectName := h.minio.ObjectName(userID, source.ID, meta.Name)
	buf := &bytes.Buffer{}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, "failed to receive chunk")
		}

		chunk := msg.GetChunk()
		if chunk == nil {
			return status.Error(codes.InvalidArgument, "expected chunk")
		}

		buf.Write(chunk)
	}

	contentType := sourceTypeToContentType(source.Type)
	if err := h.minio.Upload(stream.Context(), objectName, buf, int64(buf.Len()), contentType); err != nil {
		h.log.Error("failed to upload to minio", slog.Any("error", err))
		return status.Error(codes.Internal, "failed to upload file")
	}

	// 4. Обновляем minio_path
	source.MinioPath = objectName
	jobID := fmt.Sprintf("job-%d", source.ID) // временно, заменим на Jobs Service

	if err := h.repo.UpdateStatus(stream.Context(), source.ID, model.StatusPending, jobID); err != nil {
		h.log.Error("failed to update source", slog.Any("error", err))
		return status.Error(codes.Internal, "failed to update source")
	}

	// 5. Публикуем в Kafka
	if err := h.producer.PublishChunkingJob(stream.Context(), kafka.ChunkingJob{
		SourceID:  source.ID,
		ProjectID: source.ProjectID,
		OwnerID:   userID,
		MinioPath: objectName,
		FileType:  string(source.Type),
		JobID:     jobID,
	}); err != nil {
		h.log.Error("failed to publish kafka job", slog.Any("error", err))
		return status.Error(codes.Internal, "failed to queue processing")
	}

	return stream.SendAndClose(&pb.UploadSourceResponse{
		SourceId: source.ID,
		JobId:    jobID,
	})
}

func (h *SourceHandler) ListSources(ctx context.Context, req *pb.ListSourcesRequest) (*pb.ListSourcesResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	sources, err := h.repo.FindAll(ctx, req.ProjectId, userID)
	if err != nil {
		h.log.Error("failed to list sources", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	result := make([]*pb.Source, len(sources))
	for i, s := range sources {
		result[i] = s.ToProto()
	}

	return &pb.ListSourcesResponse{
		Sources: result,
		Total:   int32(len(sources)),
	}, nil
}

func (h *SourceHandler) GetSource(ctx context.Context, req *pb.GetSourceRequest) (*pb.SourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	source, err := h.repo.Find(ctx, req.Id, req.ProjectId, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "source not found")
	}
	if err != nil {
		h.log.Error("failed to get source", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.SourceResponse{Source: source.ToProto()}, nil
}

func (h *SourceHandler) DeleteSource(ctx context.Context, req *pb.DeleteSourceRequest) (*pb.DeleteSourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	source, err := h.repo.Find(ctx, req.Id, req.ProjectId, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "source not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Удаляем из MinIO
	if err := h.minio.Delete(ctx, source.MinioPath); err != nil {
		h.log.Error("failed to delete from minio", slog.Any("error", err))
	}

	if err := h.repo.Delete(ctx, req.Id, req.ProjectId, userID); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.DeleteSourceResponse{Success: true}, nil
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
