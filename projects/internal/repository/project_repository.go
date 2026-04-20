package repository

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/projects/internal/dto"
	"github.com/rozoomcool/sihkaromicro/projects/internal/model"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	Save(ctx context.Context, project *model.Project) error
	Find(ctx context.Context, ownerID string, projectID int64) (*model.Project, error)
	Page(ctx context.Context, ownerID string, page, pageSize int) (*dto.PageWrapper[model.Project], error)
	Delete(ctx context.Context, ownerID string, projectID int64) error
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db}
}

// Save implements ProjectRepository.
func (p *projectRepository) Save(ctx context.Context, project *model.Project) error {
	return p.db.WithContext(ctx).Create(project).Error
}

// Find implements ProjectRepository.
func (p *projectRepository) Find(ctx context.Context, ownerID string, projectID int64) (*model.Project, error) {
	var project model.Project
	err := p.db.WithContext(ctx).Where("id = ? AND owner_id = ?", projectID, ownerID).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// Page implements ProjectRepository.
func (p *projectRepository) Page(ctx context.Context, ownerID string, page int, pageSize int) (*dto.PageWrapper[model.Project], error) {
	var projects []model.Project
	var total int64

	offset := (page - 1) * pageSize

	err := p.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("owner_id = ?", ownerID).
		Count(&total).Error

	if err != nil {
		return nil, err
	}

	err = p.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Offset(offset).
		Limit(pageSize).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	return &dto.PageWrapper[model.Project]{
		Total: total,
		Data:  projects,
	}, nil
}

func (p *projectRepository) Delete(ctx context.Context, ownerID string, projectID int64) error {
	result := p.db.WithContext(ctx).
		Where("owner_id = ? AND id = ?", ownerID, projectID).
		Delete(&model.Project{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
