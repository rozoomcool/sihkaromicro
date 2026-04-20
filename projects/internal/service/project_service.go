package service

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/projects/internal/dto"
	"github.com/rozoomcool/sihkaromicro/projects/internal/model"
	"github.com/rozoomcool/sihkaromicro/projects/internal/repository"
)

type ProjectService interface {
	Create(ctx context.Context, ownerID string, title string) (*model.Project, error)
	Get(ctx context.Context, ownerID string, projectID int64) (*model.Project, error)
	List(ctx context.Context, ownerID string, page, pageSize int) (*dto.PageWrapper[model.Project], error)
	Delete(ctx context.Context, ownerID string, projectID int64) error
	CanManage(ctx context.Context, ownerID string, projectID int64) (bool, error)
}

func NewProjectService(repo repository.ProjectRepository) ProjectService {
	return &projectService{repo: repo}
}

type projectService struct {
	repo repository.ProjectRepository
}

// CanManage implements ProjectService.
func (p *projectService) CanManage(ctx context.Context, ownerID string, projectID int64) (bool, error) {
	project, err := p.repo.Find(ctx, ownerID, projectID)
	if err != nil || project == nil {
		return false, err
	}
	return true, nil
}

// Create implements ProjectService.
func (p *projectService) Create(ctx context.Context, ownerID string, title string) (*model.Project, error) {
	project := &model.Project{
		OwnerID: ownerID,
		Title:   title,
	}

	if err := p.repo.Save(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

// Delete implements ProjectService.
func (p *projectService) Delete(ctx context.Context, ownerID string, projectID int64) error {
	return p.repo.Delete(ctx, ownerID, projectID)
}

// Get implements ProjectService.
func (p *projectService) Get(ctx context.Context, ownerID string, projectID int64) (*model.Project, error) {
	return p.repo.Find(ctx, ownerID, int64(projectID))
}

// List implements ProjectService.
func (p *projectService) List(ctx context.Context, ownerID string, page int, pageSize int) (*dto.PageWrapper[model.Project], error) {
	return p.repo.Page(ctx, ownerID, page, pageSize)
}
