package service

import (
	"context"
	"fmt"

	pb "github.com/rozoomcool/sihkaromicro/proto/projects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type ProjectsClient interface {
	CheckAccess(ctx context.Context, projectID int64) (bool, error)
}

type projectsClient struct {
	client pb.ProjectsServiceClient
}

func NewProjectsClient(addr string) (ProjectsClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to projects service: %w", err)
	}

	return &projectsClient{
		client: pb.NewProjectsServiceClient(conn),
	}, nil
}

// CheckAccess — проверяем что проект принадлежит пользователю
// Пробрасываем токен пользователя
func (c *projectsClient) CheckAccess(ctx context.Context, projectID int64) (bool, error) {
	// Пробрасываем токен из входящего контекста в исходящий
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resp, err := c.client.CheckAccess(ctx, &pb.CheckAccessRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return false, fmt.Errorf("check access: %w", err)
	}

	return resp.HasAccess, nil
}
