package handler

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"

	pb "github.com/rozoomcool/sihkaromicro/proto/sources"
	"github.com/rozoomcool/sihkaromicro/sources/internal/apperr"
	"github.com/rozoomcool/sihkaromicro/sources/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"github.com/rozoomcool/sihkaromicro/sources/internal/service"
	"github.com/rozoomcool/sihkaromicro/sources/pkg/logger/sl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxFileSize = service.MaxFileSize

var allowedContentTypes = map[string]bool{
	"application/pdf": true,
	"text/plain":      true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/markdown": true,
}

type SourceHandler struct {
	pb.UnimplementedSourcesServiceServer
	srv service.SourceService
	log *slog.Logger
}

func NewSourceHandler(srv service.SourceService, log *slog.Logger) *SourceHandler {
	return &SourceHandler{srv: srv, log: log}
}

func (h *SourceHandler) Register(server *grpc.Server) {
	pb.RegisterSourcesServiceServer(server, h)
}

func (h *SourceHandler) UploadSource(stream pb.SourcesService_UploadSourceServer) error {
	ctx := stream.Context()
	op := "SourceHandler.UploadSource"
	userID := interceptor.MustUserIDFromCtx(ctx)

	log := h.log.With(slog.String("op", op), slog.String("userID", userID))

	first, err := stream.Recv()
	if err != nil {
		log.Error("failed to receive metadata", sl.Err(err))
		return status.Error(codes.InvalidArgument, "failed to receive metadata")
	}

	meta := first.GetMeta()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must contain metadata")
	}
	if meta.ProjectId <= 0 {
		return status.Error(codes.InvalidArgument, "project_id is required")
	}
	if meta.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if meta.Type == pb.SourceType_SOURCE_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "source type is required")
	}

	req := service.UploadSourceRequest{
		ProjectID: meta.ProjectId,
		OwnerID:   userID,
		Name:      meta.Name,
		Type:      protoTypeToModel(meta.Type),
		Size:      meta.Size,
		SourceURL: meta.SourceUrl,
	}

	if meta.Type == pb.SourceType_SOURCE_TYPE_URL {
		if meta.SourceUrl == "" {
			return status.Error(codes.InvalidArgument, "source_url is required for URL type")
		}
		if err := drainStream(stream); err != nil {
			return err
		}
	} else {
		if meta.Size <= 0 {
			return status.Error(codes.InvalidArgument, "size is required for file upload")
		}
		buf, contentType, err := receiveChunks(stream)
		if err != nil {
			log.Error("error while receiving chunks", sl.Err(err))
			return err
		}
		req.Reader = buf
		req.ContentType = contentType
	}

	result, err := h.srv.UploadSource(ctx, req)
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

	jobID, err := h.srv.RetryJob(ctx, req.SourceId, req.ProjectId, userID)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}

	return &pb.RetryJobResponse{JobId: jobID}, nil
}

func (h *SourceHandler) GetSource(ctx context.Context, req *pb.GetSourceRequest) (*pb.SourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	detail, err := h.srv.GetSource(ctx, req.Id, req.ProjectId, userID)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}

	proto := detail.Source.ToProto()
	proto.Url = detail.SignedURL

	return &pb.SourceResponse{Source: proto}, nil
}

func (h *SourceHandler) ListSources(ctx context.Context, req *pb.ListSourcesRequest) (*pb.ListSourcesResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	details, err := h.srv.ListSources(ctx, req.ProjectId, userID)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}

	result := make([]*pb.Source, len(details))
	for i, d := range details {
		proto := d.Source.ToProto()
		proto.Url = d.SignedURL
		result[i] = proto
	}

	return &pb.ListSourcesResponse{Sources: result}, nil
}

func (h *SourceHandler) DeleteSource(ctx context.Context, req *pb.DeleteSourceRequest) (*pb.DeleteSourceResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)

	if err := h.srv.DeleteSource(ctx, req.Id, req.ProjectId, userID); err != nil {
		return nil, apperr.ToGRPC(err)
	}

	return &pb.DeleteSourceResponse{Success: true}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

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
				return nil, "", status.Error(codes.InvalidArgument, "first chunk too small for content-type detection")
			}
			contentType = http.DetectContentType(chunk)
			if !allowedContentTypes[contentType] {
				return nil, "", status.Errorf(codes.InvalidArgument, "unsupported file type: %s", contentType)
			}
			firstChunk = false
		}

		totalSize += int64(len(chunk))
		if totalSize > maxFileSize {
			return nil, "", status.Error(codes.InvalidArgument, "file exceeds maximum allowed size")
		}

		buf.Write(chunk)
	}

	if !doneReceived {
		return nil, "", status.Error(codes.Canceled, "upload incomplete: stream closed before done signal")
	}
	if buf.Len() == 0 {
		return nil, "", status.Error(codes.InvalidArgument, "empty file")
	}

	return buf, contentType, nil
}

// drainStream consumes remaining stream messages until done=true or EOF.
// Used for SOURCE_TYPE_URL where the client sends meta then immediately done.
func drainStream(stream pb.SourcesService_UploadSourceServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Error(codes.Internal, "stream error")
		}
		if msg.GetDone() {
			return nil
		}
	}
}
