package handler

import (
	"log/slog"

	pb "github.com/rozoomcool/sihkaromicro/sources/gen/proto/sources"
	"google.golang.org/grpc"
)

type SourceGRPCHandler struct {
	pb.UnimplementedSourcesServiceServer
	log *slog.Logger
}

func NewSourceGRPCHandler(log *slog.Logger) *SourceGRPCHandler {
	return &SourceGRPCHandler{log: log}
}

func (h *SourceGRPCHandler) Register(server *grpc.Server) {
	pb.RegisterSourcesServiceServer(server, h)
}
