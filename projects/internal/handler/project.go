package handler

import (
	"context"
	"log/slog"

	pb "github.com/rozoomcool/sihkaromicro/projects/gen/proto/projects"
	"github.com/rozoomcool/sihkaromicro/projects/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/projects/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProjectGRPCHandler struct {
	pb.UnimplementedProjectsServiceServer
	service service.ProjectService
	log     *slog.Logger
}

func NewProjectGRPCHandler(projectSrv service.ProjectService, log *slog.Logger) *ProjectGRPCHandler {
	return &ProjectGRPCHandler{service: projectSrv, log: log}
}

func (h *ProjectGRPCHandler) Register(server *grpc.Server) {
	pb.RegisterProjectsServiceServer(server, h)
}

func (p *ProjectGRPCHandler) CheckAccess(ctx context.Context, in *pb.CheckAccessRequest) (*pb.CheckAccessResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)
	ok, err := p.service.CanManage(ctx, userID, in.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad request: "+err.Error())
	}
	return &pb.CheckAccessResponse{HasAccess: ok}, nil
}

func (p *ProjectGRPCHandler) CreateProject(context.Context, *pb.CreateProjectRequest) (*pb.ProjectResponse, error) {
	panic("")
}

func (p *ProjectGRPCHandler) DeleteProject(context.Context, *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	panic("")
}

func (p *ProjectGRPCHandler) GetProject(context.Context, *pb.GetProjectRequest) (*pb.ProjectResponse, error) {
	panic("")
}

func (p *ProjectGRPCHandler) ListProjects(context.Context, *pb.ListProjectsRequest) (*pb.ListProjectsResponse, error) {
	panic("")
}
