package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rozoomcool/sihkaromicro/projects/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/projects/internal/service"
	pb "github.com/rozoomcool/sihkaromicro/proto/projects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pb.CheckAccessResponse{HasAccess: false}, nil
		}
		return nil, status.Error(codes.InvalidArgument, "bad request: "+err.Error())
	}
	return &pb.CheckAccessResponse{HasAccess: ok}, nil
}

func (p *ProjectGRPCHandler) CreateProject(ctx context.Context, in *pb.CreateProjectRequest) (*pb.ProjectResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)
	project, err := p.service.Create(ctx, userID, in.Title)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Bad request: "+err.Error())
	}
	return &pb.ProjectResponse{
		Project: project.ToProto(),
	}, nil
}

func (p *ProjectGRPCHandler) DeleteProject(ctx context.Context, in *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)
	err := p.service.Delete(ctx, userID, in.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Bad request: "+err.Error())
	}
	return &pb.DeleteProjectResponse{Success: true}, nil

}

func (p *ProjectGRPCHandler) GetProject(ctx context.Context, in *pb.GetProjectRequest) (*pb.ProjectResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)
	project, err := p.service.Get(ctx, userID, in.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Bad request: "+err.Error())
	}
	return &pb.ProjectResponse{Project: project.ToProto()}, nil
}

func (p *ProjectGRPCHandler) ListProjects(ctx context.Context, in *pb.ListProjectsRequest) (*pb.ListProjectsResponse, error) {
	userID := interceptor.MustUserIDFromCtx(ctx)
	page, err := p.service.List(ctx, userID, int(in.Page), int(in.PageSize))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Bad request: "+err.Error())
	}
	projects := make([]*pb.Project, len(page.Data))
	for i, v := range page.Data {
		projects[i] = v.ToProto()
	}
	return &pb.ListProjectsResponse{Total: int32(page.Total), Projects: projects}, nil
}
