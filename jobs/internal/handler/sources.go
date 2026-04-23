package handler

import (
	"log/slog"

	pb "github.com/rozoomcool/sihkaromicro/jobs/gen/proto/jobs"
	"google.golang.org/grpc"
)

type SourceGRPCHandler struct {
	pb.UnimplementedJobsServiceServer
	log *slog.Logger
}

func NewSourceGRPCHandler(log *slog.Logger) *SourceGRPCHandler {
	return &SourceGRPCHandler{log: log}
}

func (h *SourceGRPCHandler) Register(server *grpc.Server) {
	pb.RegisterJobsServiceServer(server, h)
}
